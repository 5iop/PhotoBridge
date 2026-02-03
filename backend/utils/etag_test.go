package utils

import (
	"testing"
	"time"
)

func TestGenerateETag(t *testing.T) {
	tests := []struct {
		name      string
		photoID   uint
		updatedAt time.Time
		size      string
		wantLen   int
	}{
		{
			name:      "small thumbnail",
			photoID:   1,
			updatedAt: time.Unix(1234567890, 0),
			size:      "small",
			wantLen:   34, // "xxxx..." with quotes
		},
		{
			name:      "large thumbnail",
			photoID:   999,
			updatedAt: time.Unix(1234567890, 0),
			size:      "large",
			wantLen:   34,
		},
		{
			name:      "different photo same time",
			photoID:   2,
			updatedAt: time.Unix(1234567890, 0),
			size:      "small",
			wantLen:   34,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			etag := GenerateETag(tt.photoID, tt.updatedAt, tt.size)

			// Check format: should start and end with quotes
			if etag[0] != '"' || etag[len(etag)-1] != '"' {
				t.Errorf("ETag should be quoted, got: %s", etag)
			}

			// Check length
			if len(etag) != tt.wantLen {
				t.Errorf("ETag length = %d, want %d", len(etag), tt.wantLen)
			}

			// Check hex characters (between quotes)
			for i := 1; i < len(etag)-1; i++ {
				c := etag[i]
				if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
					t.Errorf("ETag contains non-hex character: %c", c)
					break
				}
			}
		})
	}
}

func TestGenerateETagConsistency(t *testing.T) {
	// Same input should always produce same output
	photoID := uint(123)
	updatedAt := time.Unix(1234567890, 0)
	size := "small"

	etag1 := GenerateETag(photoID, updatedAt, size)
	etag2 := GenerateETag(photoID, updatedAt, size)

	if etag1 != etag2 {
		t.Errorf("Same input produced different ETags: %s != %s", etag1, etag2)
	}
}

func TestGenerateETagDifferentInputs(t *testing.T) {
	baseTime := time.Unix(1234567890, 0)

	tests := []struct {
		name      string
		photoID1  uint
		photoID2  uint
		time1     time.Time
		time2     time.Time
		size1     string
		size2     string
		shouldDif bool
	}{
		{
			name:      "different photo ID",
			photoID1:  1,
			photoID2:  2,
			time1:     baseTime,
			time2:     baseTime,
			size1:     "small",
			size2:     "small",
			shouldDif: true,
		},
		{
			name:      "different update time",
			photoID1:  1,
			photoID2:  1,
			time1:     baseTime,
			time2:     baseTime.Add(time.Hour),
			size1:     "small",
			size2:     "small",
			shouldDif: true,
		},
		{
			name:      "different size",
			photoID1:  1,
			photoID2:  1,
			time1:     baseTime,
			time2:     baseTime,
			size1:     "small",
			size2:     "large",
			shouldDif: true,
		},
		{
			name:      "same everything",
			photoID1:  1,
			photoID2:  1,
			time1:     baseTime,
			time2:     baseTime,
			size1:     "small",
			size2:     "small",
			shouldDif: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			etag1 := GenerateETag(tt.photoID1, tt.time1, tt.size1)
			etag2 := GenerateETag(tt.photoID2, tt.time2, tt.size2)

			if tt.shouldDif {
				if etag1 == etag2 {
					t.Errorf("Expected different ETags, got same: %s", etag1)
				}
			} else {
				if etag1 != etag2 {
					t.Errorf("Expected same ETags, got different: %s != %s", etag1, etag2)
				}
			}
		})
	}
}
