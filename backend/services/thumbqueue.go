package services

import (
	"errors"
	"log"
	"path/filepath"
	"runtime/debug"
	"sync"
	"time"

	"photobridge/config"
	"photobridge/database"
	"photobridge/models"
	"photobridge/utils"
)

const (
	shortname      = "[ThumbQueue]"
	maxQueueLength = 1000 // Limit queue length to prevent memory exhaustion
)

var ErrThumbnailTimeout = errors.New("thumbnail generation timeout")

// ThumbTask represents a thumbnail generation task (only stores path info, not image data)
type ThumbTask struct {
	PhotoID     uint
	ProjectName string
	BaseName    string
	NormalExt   string
}

// ThumbQueue manages thumbnail generation with an unbounded queue
type ThumbQueue struct {
	tasks      []ThumbTask
	tasksMu    sync.Mutex
	cond       *sync.Cond
	processing sync.Map // Track which photos are being processed or queued
	workers    int
	jobTimeout time.Duration
	running    bool
	stopCh     chan struct{}
	wg         sync.WaitGroup
}

var (
	// Queue is the global thumbnail queue instance
	Queue *ThumbQueue
)

// InitQueue initializes the global thumbnail queue
func InitQueue(workers int, jobTimeout time.Duration) {
	q := &ThumbQueue{
		tasks:      make([]ThumbTask, 0),
		workers:    workers,
		jobTimeout: jobTimeout,
		stopCh:     make(chan struct{}),
	}
	q.cond = sync.NewCond(&q.tasksMu)
	q.Start()
	Queue = q
	log.Printf("%s Initialized with %d workers, timeout=%s", shortname, workers, jobTimeout)
}

// Start begins the worker goroutines
func (q *ThumbQueue) Start() {
	q.tasksMu.Lock()
	if q.running {
		q.tasksMu.Unlock()
		return
	}
	q.running = true
	q.tasksMu.Unlock()

	for i := 0; i < q.workers; i++ {
		q.wg.Add(1)
		go q.worker(i)
	}
}

// worker processes tasks from the queue
func (q *ThumbQueue) worker(id int) {
	defer q.wg.Done()
	log.Printf("%s Worker %d started", shortname, id)

	for {
		// Get next task
		q.tasksMu.Lock()
		for len(q.tasks) == 0 && q.running {
			q.cond.Wait()
		}

		if !q.running && len(q.tasks) == 0 {
			q.tasksMu.Unlock()
			break
		}

		// Pop task from front
		task := q.tasks[0]
		q.tasks = q.tasks[1:]
		q.tasksMu.Unlock()

		// Process task
		q.processTaskSafely(task, id)
	}

	log.Printf("%s Worker %d stopped", shortname, id)
}

// processTaskSafely ensures a panic in one task does not kill the worker.
func (q *ThumbQueue) processTaskSafely(task ThumbTask, workerID int) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("%s Worker %d panic while processing photo %d: %v\n%s",
				shortname, workerID, task.PhotoID, r, string(debug.Stack()))
			q.processing.Delete(task.PhotoID)
		}
	}()
	q.processTask(task)
}

// processTask generates thumbnails for a single photo from file path
func (q *ThumbQueue) processTask(task ThumbTask) {
	defer q.processing.Delete(task.PhotoID)

	if task.NormalExt == "" {
		return // Only RAW, skip
	}

	// Validate project name for path safety
	if !utils.ValidatePathComponent(task.ProjectName) {
		log.Printf("%s Invalid project name for photo %d: %s", shortname, task.PhotoID, task.ProjectName)
		return
	}

	// Generate thumbnail from file path (not from memory)
	imagePath := filepath.Join(config.AppConfig.UploadDir, task.ProjectName, task.BaseName+task.NormalExt)

	// Validate the image path is secure
	safeImagePath, err := utils.ValidateSecurePath(config.AppConfig.UploadDir, imagePath)
	if err != nil {
		log.Printf("%s Invalid file path for photo %d: %v", shortname, task.PhotoID, err)
		return
	}

	thumbResult, err := q.generateWithTimeout(safeImagePath)
	if err != nil {
		log.Printf("%s Failed to generate thumbnail for photo %d (%s): %v", shortname, task.PhotoID, safeImagePath, err)
		return
	}

	// Update database
	if err := database.DB.Model(&models.Photo{}).Where("id = ?", task.PhotoID).Updates(map[string]interface{}{
		"thumb_small":  thumbResult.Small,
		"thumb_large":  thumbResult.Large,
		"thumb_width":  thumbResult.Width,
		"thumb_height": thumbResult.Height,
	}).Error; err != nil {
		log.Printf("%s Failed to save thumbnail for photo %d: %v", shortname, task.PhotoID, err)
		return
	}

	log.Printf("%s Generated thumbnail for photo %d", shortname, task.PhotoID)
}

func (q *ThumbQueue) generateWithTimeout(imagePath string) (*utils.ThumbnailResult, error) {
	if q.jobTimeout <= 0 {
		return utils.GenerateThumbnails(imagePath)
	}

	type thumbResult struct {
		result *utils.ThumbnailResult
		err    error
	}
	done := make(chan thumbResult, 1)
	go func() {
		result, err := utils.GenerateThumbnails(imagePath)
		done <- thumbResult{result: result, err: err}
	}()

	select {
	case out := <-done:
		return out.result, out.err
	case <-time.After(q.jobTimeout):
		return nil, ErrThumbnailTimeout
	}
}

// Enqueue adds a thumbnail generation task to the queue
// Returns true if the task was added, false if it's already queued or processing
func (q *ThumbQueue) Enqueue(photo *models.Photo, projectName string) bool {
	if photo.NormalExt == "" {
		return false // Only RAW, no thumbnail needed
	}

	// Check if already queued or processing
	if _, loaded := q.processing.LoadOrStore(photo.ID, true); loaded {
		return false // Already in queue or processing
	}

	task := ThumbTask{
		PhotoID:     photo.ID,
		ProjectName: projectName,
		BaseName:    photo.BaseName,
		NormalExt:   photo.NormalExt,
	}

	q.tasksMu.Lock()
	// Reject enqueue when queue is stopped to avoid "stuck processing" states.
	if !q.running {
		q.tasksMu.Unlock()
		q.processing.Delete(photo.ID)
		return false
	}

	// Check queue length limit to prevent memory exhaustion
	if len(q.tasks) >= maxQueueLength {
		q.tasksMu.Unlock()
		q.processing.Delete(photo.ID) // Remove from processing map
		log.Printf("%s Queue full (%d), rejecting photo %d", shortname, maxQueueLength, photo.ID)
		return false
	}

	q.tasks = append(q.tasks, task)
	queueLen := len(q.tasks)
	q.cond.Signal() // Wake up one worker
	q.tasksMu.Unlock()

	log.Printf("%s Enqueued photo %d (queue length: %d)", shortname, photo.ID, queueLen)
	return true
}

// IsRunning reports whether the queue is accepting and processing tasks.
func (q *ThumbQueue) IsRunning() bool {
	q.tasksMu.Lock()
	defer q.tasksMu.Unlock()
	return q.running
}

// EnqueueByID adds a photo to the queue by its ID
func (q *ThumbQueue) EnqueueByID(photoID uint) bool {
	var photo models.Photo
	if err := database.DB.First(&photo, photoID).Error; err != nil {
		return false
	}

	var project models.Project
	if err := database.DB.First(&project, photo.ProjectID).Error; err != nil {
		return false
	}

	return q.Enqueue(&photo, project.Name)
}

// QueueLength returns the current number of tasks in the queue
func (q *ThumbQueue) QueueLength() int {
	q.tasksMu.Lock()
	defer q.tasksMu.Unlock()
	return len(q.tasks)
}

// IsProcessing checks if a photo is being processed or queued
func (q *ThumbQueue) IsProcessing(photoID uint) bool {
	_, exists := q.processing.Load(photoID)
	return exists
}

// Stop gracefully stops the queue (waits for current tasks to complete)
func (q *ThumbQueue) Stop() {
	q.tasksMu.Lock()
	q.running = false
	q.cond.Broadcast() // Wake up all workers
	q.tasksMu.Unlock()

	q.wg.Wait()
	log.Printf("%s Queue stopped", shortname)
}
