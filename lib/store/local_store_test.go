package store

import (
	"fmt"
	"os"
	"path"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileHashStates(t *testing.T) {
	s, cleanup := LocalStoreWithRefcountFixture()
	defer cleanup()

	s.CreateUploadFile("test_file.txt", 100)
	err := s.SetUploadFileHashState("test_file.txt", []byte{uint8(0), uint8(1)}, "sha256", "500")
	assert.Nil(t, err)
	b, err := s.GetUploadFileHashState("test_file.txt", "sha256", "500")
	assert.Nil(t, err)
	assert.Equal(t, uint8(0), b[0])
	assert.Equal(t, uint8(1), b[1])

	l, err := s.ListUploadFileHashStatePaths("test_file.txt")
	assert.Nil(t, err)
	assert.Equal(t, len(l), 1)
	assert.True(t, strings.HasSuffix(l[0], "/hashstates/sha256/500"))
}

func TestCreateUploadFileAndMoveToCache(t *testing.T) {
	s, cleanup := LocalStoreWithRefcountFixture()
	defer cleanup()

	err := s.CreateUploadFile("test_file.txt", 100)
	assert.Nil(t, err)
	err = s.SetUploadFileHashState("test_file.txt", []byte{uint8(0), uint8(1)}, "sha256", "500")
	assert.Nil(t, err)
	b, err := s.GetUploadFileHashState("test_file.txt", "sha256", "500")
	assert.Nil(t, err)
	assert.Equal(t, uint8(0), b[0])
	assert.Equal(t, uint8(1), b[1])
	err = s.SetUploadFileStartedAt("test_file.txt", []byte{uint8(2), uint8(3)})
	assert.Nil(t, err)
	b, err = s.GetUploadFileStartedAt("test_file.txt")
	assert.Nil(t, err)
	assert.Equal(t, uint8(2), b[0])
	assert.Equal(t, uint8(3), b[1])
	_, err = os.Stat(path.Join(s.Config().UploadDir, "te", "st", "test_file.txt"))
	assert.Nil(t, err)

	err = s.MoveUploadFileToCache("test_file.txt", "test_file_cache.txt")
	assert.Nil(t, err)
	_, err = os.Stat(path.Join(s.Config().UploadDir, "te", "st", "test_file.txt"))
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(path.Join(s.Config().CacheDir, "te", "st", "test_file_cache.txt"))
	assert.Nil(t, err)
}

func TestDownloadAndDeleteFiles(t *testing.T) {
	s, cleanup := LocalStoreWithRefcountFixture()
	defer cleanup()

	var waitGroup sync.WaitGroup

	for i := 0; i < 100; i++ {
		waitGroup.Add(1)

		testFileName := fmt.Sprintf("test_%d", i)
		go func() {
			err := s.CreateDownloadFile(testFileName, 1)
			assert.Nil(t, err)
			err = s.MoveDownloadFileToCache(testFileName)
			assert.Nil(t, err)
			err = s.MoveCacheFileToTrash(testFileName)
			assert.Nil(t, err)

			waitGroup.Done()
		}()
	}

	waitGroup.Wait()
	err := s.DeleteAllTrashFiles()
	assert.Nil(t, err)

	for i := 0; i < 100; i++ {
		testFileName := fmt.Sprintf("test_%d", i)
		_, err := os.Stat(path.Join(s.Config().TrashDir, testFileName))
		assert.True(t, os.IsNotExist(err))
	}
}

func TestTrashDeletionCronDeletesFiles(t *testing.T) {
	require := require.New(t)

	interval := time.Second

	s, cleanup := LocalStoreWithTrashDeletionFixture(interval)
	defer cleanup()

	f := "test_file.txt"
	require.NoError(s.CreateDownloadFile(f, 1))
	require.NoError(s.MoveDownloadOrCacheFileToTrash(f))

	time.Sleep(interval + 250*time.Millisecond)

	_, err := os.Stat(path.Join(s.Config().TrashDir, f))
	require.True(os.IsNotExist(err))
}