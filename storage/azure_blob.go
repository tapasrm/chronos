package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blockblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
)

type AzureBlobStorage struct {
	containerClient *container.Client
	cdnBaseURL      string
}

func NewAzureBlobStorage(accountName, accountKey, containerName, cdnBaseURL string) (*AzureBlobStorage, error) {
	cred, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		return nil, err
	}

	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", accountName)
	client, err := azblob.NewClientWithSharedKeyCredential(serviceURL, cred, nil)
	if err != nil {
		return nil, err
	}
	serviceClient := client.ServiceClient()
	containerClient := serviceClient.NewContainerClient(containerName)
	return &AzureBlobStorage{
		containerClient: containerClient,
		cdnBaseURL:      cdnBaseURL,
	}, nil
}

func (s *AzureBlobStorage) DownloadFile(ctx context.Context, name string) (io.ReadCloser, error) {
	blob := s.containerClient.NewBlockBlobClient(name)
	resp, err := blob.DownloadStream(ctx, nil)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

func (s *AzureBlobStorage) ListFiles(ctx context.Context) ([]FileInfo, error) {
	pager := s.containerClient.NewListBlobsFlatPager(nil)
	var files []FileInfo
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, blob := range page.Segment.BlobItems {
			files = append(files, FileInfo{
				Name: *blob.Name,
				URL:  fmt.Sprintf("%s/%s", s.cdnBaseURL, *blob.Name),
			})
		}
	}
	return files, nil
}

func (s *AzureBlobStorage) UploadFile(ctx context.Context, name string, data io.Reader) (FileInfo, error) {
	blobClient := s.containerClient.NewBlockBlobClient(name)
	_, err := blobClient.UploadStream(ctx, data, &blockblob.UploadStreamOptions{})
	if err != nil {
		return FileInfo{}, err
	}
	return FileInfo{
		Name: name,
		URL:  fmt.Sprintf("%s/%s", s.cdnBaseURL, name),
	}, nil
}

func (s *AzureBlobStorage) DeleteFile(ctx context.Context, name string) error {
	blobClient := s.containerClient.NewBlobClient(name)
	_, err := blobClient.Delete(ctx, nil)
	return err
}

func (s *AzureBlobStorage) RenameFile(ctx context.Context, oldName, newName string) error {
	oldBlob := s.containerClient.NewBlobClient(oldName)
	newBlob := s.containerClient.NewBlockBlobClient(newName)

	_, err := newBlob.StartCopyFromURL(ctx, oldBlob.URL(), nil)
	if err != nil {
		return err
	}

	_, err = oldBlob.Delete(ctx, nil)
	return err
}
