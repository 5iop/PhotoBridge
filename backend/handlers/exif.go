package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"photobridge/config"
	"photobridge/database"
	"photobridge/models"

	"github.com/gin-gonic/gin"
	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/tiff"
)

type ExifInfo struct {
	CameraMake    string `json:"camera_make,omitempty"`
	CameraModel   string `json:"camera_model,omitempty"`
	LensMake      string `json:"lens_make,omitempty"`
	LensModel     string `json:"lens_model,omitempty"`
	FocalLength   string `json:"focal_length,omitempty"`
	Aperture      string `json:"aperture,omitempty"`
	ShutterSpeed  string `json:"shutter_speed,omitempty"`
	ISO           string `json:"iso,omitempty"`
	DateTime      string `json:"date_time,omitempty"`
	Width         int    `json:"width,omitempty"`
	Height        int    `json:"height,omitempty"`
	Orientation   string `json:"orientation,omitempty"`
	ExposureMode  string `json:"exposure_mode,omitempty"`
	WhiteBalance  string `json:"white_balance,omitempty"`
	Flash         string `json:"flash,omitempty"`
	MeteringMode  string `json:"metering_mode,omitempty"`
	Software      string `json:"software,omitempty"`
	GPSLatitude   string `json:"gps_latitude,omitempty"`
	GPSLongitude  string `json:"gps_longitude,omitempty"`
}

func getTagString(x *exif.Exif, tag exif.FieldName) string {
	t, err := x.Get(tag)
	if err != nil {
		return ""
	}
	return t.String()
}

func getTagStringVal(x *exif.Exif, tag exif.FieldName) string {
	t, err := x.Get(tag)
	if err != nil {
		return ""
	}
	if t.Format() == tiff.StringVal {
		s, _ := t.StringVal()
		return s
	}
	return t.String()
}

func getTagInt(x *exif.Exif, tag exif.FieldName) int {
	t, err := x.Get(tag)
	if err != nil {
		return 0
	}
	i, _ := t.Int(0)
	return i
}

func formatRational(tag *tiff.Tag) string {
	if tag == nil {
		return ""
	}
	num, denom, err := tag.Rat2(0)
	if err != nil {
		return tag.String()
	}
	if denom == 0 {
		return ""
	}
	// Check if it's a simple integer
	if num%denom == 0 {
		return fmt.Sprintf("%d", num/denom)
	}
	// Return as decimal
	return fmt.Sprintf("%.1f", float64(num)/float64(denom))
}

func GetPhotoExif(c *gin.Context) {
	token := c.Param("token")
	photoID := c.Param("photoId")

	var link models.ShareLink
	result := database.DB.Where("token = ?", token).Preload("Exclusions").First(&link)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Share link not found"})
		return
	}

	// Check if photo is excluded
	for _, e := range link.Exclusions {
		if fmt.Sprintf("%d", e.PhotoID) == photoID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Photo not accessible"})
			return
		}
	}

	var photo models.Photo
	if err := database.DB.First(&photo, photoID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Photo not found"})
		return
	}

	var project models.Project
	database.DB.First(&project, photo.ProjectID)

	// Find the file to read EXIF from - prioritize RAW files
	var x *exif.Exif

	// Try RAW file first if available
	if photo.HasRaw && photo.RawExt != "" {
		rawPath := filepath.Join(config.AppConfig.UploadDir, project.Name, photo.BaseName+photo.RawExt)
		if f, openErr := os.Open(rawPath); openErr == nil {
			// 使用闭包确保文件正确关闭，即使Decode失败
			func() {
				defer f.Close()
				x, _ = exif.Decode(f)
			}()
		}
	}

	// If RAW failed or not available, try normal image file
	if x == nil && photo.NormalExt != "" {
		normalPath := filepath.Join(config.AppConfig.UploadDir, project.Name, photo.BaseName+photo.NormalExt)
		f, openErr := os.Open(normalPath)
		if openErr != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
			return
		}
		defer f.Close()

		var decodeErr error
		x, decodeErr = exif.Decode(f)
		if decodeErr != nil {
			// Return empty EXIF info if can't decode
			c.JSON(http.StatusOK, ExifInfo{})
			return
		}
	}

	// No file found or EXIF decoded
	if x == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No image file found or EXIF data unavailable"})
		return
	}

	info := ExifInfo{}

	// Camera info
	info.CameraMake = getTagStringVal(x, exif.Make)
	info.CameraModel = getTagStringVal(x, exif.Model)
	info.LensModel = getTagStringVal(x, exif.LensModel)
	info.Software = getTagStringVal(x, exif.Software)

	// Focal length
	if tag, err := x.Get(exif.FocalLength); err == nil {
		info.FocalLength = formatRational(tag) + "mm"
	}

	// Aperture
	if tag, err := x.Get(exif.FNumber); err == nil {
		info.Aperture = "f/" + formatRational(tag)
	}

	// Shutter speed
	if tag, err := x.Get(exif.ExposureTime); err == nil {
		num, denom, err := tag.Rat2(0)
		if err == nil && denom != 0 {
			if num < denom {
				info.ShutterSpeed = fmt.Sprintf("%d/%d s", num, denom)
			} else {
				info.ShutterSpeed = fmt.Sprintf("%.1f s", float64(num)/float64(denom))
			}
		}
	}

	// ISO
	if tag, err := x.Get(exif.ISOSpeedRatings); err == nil {
		iso, _ := tag.Int(0)
		info.ISO = fmt.Sprintf("ISO %d", iso)
	}

	// Date/Time
	if tm, err := x.DateTime(); err == nil {
		info.DateTime = tm.Format("2006-01-02 15:04:05")
	}

	// Image dimensions
	info.Width = getTagInt(x, exif.PixelXDimension)
	info.Height = getTagInt(x, exif.PixelYDimension)

	// Orientation
	if orient := getTagInt(x, exif.Orientation); orient > 0 {
		orientations := map[int]string{
			1: "Normal",
			2: "Flip Horizontal",
			3: "Rotate 180",
			4: "Flip Vertical",
			5: "Transpose",
			6: "Rotate 90 CW",
			7: "Transverse",
			8: "Rotate 90 CCW",
		}
		info.Orientation = orientations[orient]
	}

	// Exposure mode
	if mode := getTagInt(x, exif.ExposureMode); mode >= 0 {
		modes := map[int]string{
			0: "Auto",
			1: "Manual",
			2: "Auto Bracket",
		}
		if m, ok := modes[mode]; ok {
			info.ExposureMode = m
		}
	}

	// White balance
	if wb := getTagInt(x, exif.WhiteBalance); wb >= 0 {
		wbs := map[int]string{
			0: "Auto",
			1: "Manual",
		}
		if w, ok := wbs[wb]; ok {
			info.WhiteBalance = w
		}
	}

	// Flash
	if flash := getTagInt(x, exif.Flash); flash >= 0 {
		if flash&1 == 1 {
			info.Flash = "Fired"
		} else {
			info.Flash = "No Flash"
		}
	}

	// Metering mode
	if meter := getTagInt(x, exif.MeteringMode); meter > 0 {
		meters := map[int]string{
			1: "Average",
			2: "Center Weighted",
			3: "Spot",
			4: "Multi Spot",
			5: "Pattern",
			6: "Partial",
		}
		if m, ok := meters[meter]; ok {
			info.MeteringMode = m
		}
	}

	// GPS
	lat, lng, err := x.LatLong()
	if err == nil {
		info.GPSLatitude = fmt.Sprintf("%.6f", lat)
		info.GPSLongitude = fmt.Sprintf("%.6f", lng)
	}

	c.JSON(http.StatusOK, info)
}

// GetAdminPhotoExif - for admin panel
func GetAdminPhotoExif(c *gin.Context) {
	photoID := c.Param("id")

	var photo models.Photo
	if err := database.DB.First(&photo, photoID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Photo not found"})
		return
	}

	var project models.Project
	database.DB.First(&project, photo.ProjectID)

	// Find the file to read EXIF from - prioritize RAW files
	var x *exif.Exif

	// Try RAW file first if available
	if photo.HasRaw && photo.RawExt != "" {
		rawPath := filepath.Join(config.AppConfig.UploadDir, project.Name, photo.BaseName+photo.RawExt)
		if f, openErr := os.Open(rawPath); openErr == nil {
			// 使用闭包确保文件正确关闭，即使Decode失败
			func() {
				defer f.Close()
				x, _ = exif.Decode(f)
			}()
		}
	}

	// If RAW failed or not available, try normal image file
	if x == nil && photo.NormalExt != "" {
		normalPath := filepath.Join(config.AppConfig.UploadDir, project.Name, photo.BaseName+photo.NormalExt)
		f, openErr := os.Open(normalPath)
		if openErr != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
			return
		}
		defer f.Close()

		var decodeErr error
		x, decodeErr = exif.Decode(f)
		if decodeErr != nil {
			c.JSON(http.StatusOK, ExifInfo{})
			return
		}
	}

	// No file found or EXIF decoded
	if x == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No image file found or EXIF data unavailable"})
		return
	}

	info := ExifInfo{}

	info.CameraMake = getTagStringVal(x, exif.Make)
	info.CameraModel = getTagStringVal(x, exif.Model)
	info.LensModel = getTagStringVal(x, exif.LensModel)
	info.Software = getTagStringVal(x, exif.Software)

	if tag, err := x.Get(exif.FocalLength); err == nil {
		info.FocalLength = formatRational(tag) + "mm"
	}

	if tag, err := x.Get(exif.FNumber); err == nil {
		info.Aperture = "f/" + formatRational(tag)
	}

	if tag, err := x.Get(exif.ExposureTime); err == nil {
		num, denom, err := tag.Rat2(0)
		if err == nil && denom != 0 {
			if num < denom {
				info.ShutterSpeed = fmt.Sprintf("%d/%d s", num, denom)
			} else {
				info.ShutterSpeed = fmt.Sprintf("%.1f s", float64(num)/float64(denom))
			}
		}
	}

	if tag, err := x.Get(exif.ISOSpeedRatings); err == nil {
		iso, _ := tag.Int(0)
		info.ISO = fmt.Sprintf("ISO %d", iso)
	}

	if tm, err := x.DateTime(); err == nil {
		info.DateTime = tm.Format("2006-01-02 15:04:05")
	}

	info.Width = getTagInt(x, exif.PixelXDimension)
	info.Height = getTagInt(x, exif.PixelYDimension)

	c.JSON(http.StatusOK, info)
}
