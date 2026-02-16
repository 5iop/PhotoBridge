package services

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"photobridge/models"
)

// createTestQueue creates a ThumbQueue for testing without starting workers
func createTestQueue() *ThumbQueue {
	q := &ThumbQueue{
		tasks:   make([]ThumbTask, 0),
		workers: 2,
		stopCh:  make(chan struct{}),
		running: true,
	}
	q.cond = sync.NewCond(&q.tasksMu)
	return q
}

func TestThumbQueueEnqueue(t *testing.T) {
	q := createTestQueue()

	photo := &models.Photo{
		BaseName:  "test",
		NormalExt: ".jpg",
	}
	photo.ID = 1

	// First enqueue should succeed
	result := q.Enqueue(photo, "test-project")
	if !result {
		t.Error("First enqueue should return true")
	}

	if q.QueueLength() != 1 {
		t.Errorf("Queue length should be 1, got %d", q.QueueLength())
	}

	// Second enqueue with same photo ID should fail (duplicate prevention)
	result = q.Enqueue(photo, "test-project")
	if result {
		t.Error("Second enqueue of same photo should return false")
	}

	if q.QueueLength() != 1 {
		t.Errorf("Queue length should still be 1, got %d", q.QueueLength())
	}
}

func TestThumbQueueEnqueueRawOnly(t *testing.T) {
	q := createTestQueue()

	// RAW-only photo (no NormalExt) should not be enqueued
	photo := &models.Photo{
		BaseName: "rawfile",
		RawExt:   ".cr2",
		HasRaw:   true,
	}
	photo.ID = 1

	result := q.Enqueue(photo, "test-project")
	if result {
		t.Error("RAW-only photo should not be enqueued")
	}

	if q.QueueLength() != 0 {
		t.Errorf("Queue should be empty, got %d", q.QueueLength())
	}
}

func TestThumbQueueIsProcessing(t *testing.T) {
	q := createTestQueue()

	photo := &models.Photo{
		BaseName:  "test",
		NormalExt: ".jpg",
	}
	photo.ID = 1

	// Before enqueue
	if q.IsProcessing(1) {
		t.Error("Photo should not be processing before enqueue")
	}

	q.Enqueue(photo, "test-project")

	// After enqueue
	if !q.IsProcessing(1) {
		t.Error("Photo should be processing after enqueue")
	}
}

func TestThumbQueueQueueLength(t *testing.T) {
	q := createTestQueue()

	if q.QueueLength() != 0 {
		t.Errorf("Initial queue length should be 0, got %d", q.QueueLength())
	}

	// Add multiple photos
	for i := uint(1); i <= 5; i++ {
		photo := &models.Photo{
			BaseName:  "test",
			NormalExt: ".jpg",
		}
		photo.ID = i
		q.Enqueue(photo, "test-project")
	}

	if q.QueueLength() != 5 {
		t.Errorf("Queue length should be 5, got %d", q.QueueLength())
	}
}

func TestThumbTaskFields(t *testing.T) {
	task := ThumbTask{
		PhotoID:     123,
		ProjectName: "test-project",
		BaseName:    "DSC_0001",
		NormalExt:   ".jpg",
	}

	if task.PhotoID != 123 {
		t.Errorf("PhotoID should be 123, got %d", task.PhotoID)
	}
	if task.ProjectName != "test-project" {
		t.Errorf("ProjectName should be 'test-project', got %s", task.ProjectName)
	}
	if task.BaseName != "DSC_0001" {
		t.Errorf("BaseName should be 'DSC_0001', got %s", task.BaseName)
	}
	if task.NormalExt != ".jpg" {
		t.Errorf("NormalExt should be '.jpg', got %s", task.NormalExt)
	}
}

func TestThumbQueueStartStop(t *testing.T) {
	q := createTestQueue()
	q.running = false

	// Start should set running to true
	q.Start()

	q.tasksMu.Lock()
	if !q.running {
		t.Error("Queue should be running after Start()")
	}
	q.tasksMu.Unlock()

	// Double start should be idempotent
	q.Start()
	q.tasksMu.Lock()
	if !q.running {
		t.Error("Queue should still be running after double Start()")
	}
	q.tasksMu.Unlock()

	// Stop should set running to false
	q.Stop()

	q.tasksMu.Lock()
	if q.running {
		t.Error("Queue should not be running after Stop()")
	}
	q.tasksMu.Unlock()
}

func TestThumbQueueConcurrentEnqueue(t *testing.T) {
	q := createTestQueue()

	var wg sync.WaitGroup
	var successCount int32

	// Enqueue 100 photos concurrently
	for i := uint(1); i <= 100; i++ {
		wg.Add(1)
		go func(id uint) {
			defer wg.Done()
			photo := &models.Photo{
				BaseName:  "test",
				NormalExt: ".jpg",
			}
			photo.ID = id
			if q.Enqueue(photo, "test-project") {
				atomic.AddInt32(&successCount, 1)
			}
		}(i)
	}

	wg.Wait()

	if successCount != 100 {
		t.Errorf("Expected 100 successful enqueues, got %d", successCount)
	}

	if q.QueueLength() != 100 {
		t.Errorf("Queue length should be 100, got %d", q.QueueLength())
	}
}

func TestThumbQueueDuplicatePrevention(t *testing.T) {
	q := createTestQueue()

	var wg sync.WaitGroup
	var successCount int32

	// Try to enqueue the same photo 100 times concurrently
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			photo := &models.Photo{
				BaseName:  "same-photo",
				NormalExt: ".jpg",
			}
			photo.ID = 1 // Same ID for all
			if q.Enqueue(photo, "test-project") {
				atomic.AddInt32(&successCount, 1)
			}
		}()
	}

	wg.Wait()

	// Only one should succeed
	if successCount != 1 {
		t.Errorf("Expected exactly 1 successful enqueue for duplicate photo, got %d", successCount)
	}

	if q.QueueLength() != 1 {
		t.Errorf("Queue length should be 1, got %d", q.QueueLength())
	}
}

func TestThumbQueueProcessingCleanup(t *testing.T) {
	q := createTestQueue()

	// Manually mark a photo as processing
	q.processing.Store(uint(1), true)

	if !q.IsProcessing(1) {
		t.Error("Photo 1 should be marked as processing")
	}

	// Simulate cleanup
	q.processing.Delete(uint(1))

	if q.IsProcessing(1) {
		t.Error("Photo 1 should not be processing after delete")
	}
}

func TestThumbQueueMaxLimit(t *testing.T) {
	q := createTestQueue()

	// Try to enqueue more than the max queue length (1000)
	// The queue should reject items beyond the limit
	successCount := 0
	for i := uint(1); i <= 1500; i++ {
		photo := &models.Photo{
			BaseName:  "test",
			NormalExt: ".jpg",
		}
		photo.ID = i
		if q.Enqueue(photo, "test-project") {
			successCount++
		}
	}

	// Should only accept up to maxQueueLength (1000)
	if successCount != 1000 {
		t.Errorf("Expected %d successful enqueues, got %d", 1000, successCount)
	}

	if q.QueueLength() != 1000 {
		t.Errorf("Queue length should be capped at 1000, got %d", q.QueueLength())
	}
}

func TestThumbQueueBelowLimit(t *testing.T) {
	q := createTestQueue()

	// Enqueue below the limit
	count := 500
	for i := uint(1); i <= uint(count); i++ {
		photo := &models.Photo{
			BaseName:  "test",
			NormalExt: ".jpg",
		}
		photo.ID = i
		result := q.Enqueue(photo, "test-project")
		if !result {
			t.Errorf("Enqueue should succeed when below limit, failed at %d", i)
		}
	}

	if q.QueueLength() != count {
		t.Errorf("Queue should hold %d items, got %d", count, q.QueueLength())
	}
}

func TestThumbQueueSignaling(t *testing.T) {
	q := createTestQueue()
	q.running = true

	signaled := make(chan bool, 1)

	// Goroutine waiting for signal
	go func() {
		q.tasksMu.Lock()
		for len(q.tasks) == 0 && q.running {
			q.cond.Wait()
		}
		q.tasksMu.Unlock()
		signaled <- true
	}()

	// Give the goroutine time to start waiting
	time.Sleep(50 * time.Millisecond)

	// Enqueue should signal
	photo := &models.Photo{
		BaseName:  "test",
		NormalExt: ".jpg",
	}
	photo.ID = 1
	q.Enqueue(photo, "test-project")

	select {
	case <-signaled:
		// Success
	case <-time.After(1 * time.Second):
		t.Error("Enqueue should have signaled waiting goroutine")
	}
}

func TestThumbQueueMultipleProjects(t *testing.T) {
	q := createTestQueue()

	// Enqueue photos from different projects
	photo1 := &models.Photo{BaseName: "photo1", NormalExt: ".jpg"}
	photo1.ID = 1
	photo2 := &models.Photo{BaseName: "photo2", NormalExt: ".png"}
	photo2.ID = 2

	q.Enqueue(photo1, "project-a")
	q.Enqueue(photo2, "project-b")

	if q.QueueLength() != 2 {
		t.Errorf("Queue should have 2 items, got %d", q.QueueLength())
	}
}

func TestThumbQueuePhotoWithBothFormats(t *testing.T) {
	q := createTestQueue()

	// Photo with both normal and RAW extension
	photo := &models.Photo{
		BaseName:  "DSC_0001",
		NormalExt: ".jpg",
		RawExt:    ".cr2",
		HasRaw:    true,
	}
	photo.ID = 1

	result := q.Enqueue(photo, "test-project")
	if !result {
		t.Error("Photo with NormalExt should be enqueued even if it has RAW")
	}

	if q.QueueLength() != 1 {
		t.Errorf("Queue length should be 1, got %d", q.QueueLength())
	}
}

func TestThumbQueueEmptyNormalExt(t *testing.T) {
	q := createTestQueue()

	// Photo with empty NormalExt (only RAW)
	photo := &models.Photo{
		BaseName: "DSC_0001",
		RawExt:   ".cr2",
		HasRaw:   true,
	}
	photo.ID = 1

	result := q.Enqueue(photo, "test-project")
	if result {
		t.Error("Photo without NormalExt should not be enqueued")
	}

	if q.QueueLength() != 0 {
		t.Errorf("Queue should be empty, got %d", q.QueueLength())
	}
}
