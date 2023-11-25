package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/spf13/cobra"
)

var (
	bucketName string
	outputFile string
)

type ObjectInfo struct {
	S3Bucket     string
	S3Path       string
	FileName     string
	FileSize     int64
	ModifiedDate time.Time
}

var queryCmd = &cobra.Command{
	Use:   "query",
	Short: "Recursively query an S3 bucket and output object paths",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("query called")
		objects, err := listObjects(context.TODO(), bucketName, "")
		if err != nil {
			fmt.Println("Error listing objects:", err)
			return
		}

		fileNameMap := make(map[string]ObjectInfo)

		alreadyTranscribed := genTranscriptionCompletedMap(objects)

		for _, obj := range objects {
			fileNameMap[obj.FileName] = obj
		}

		for _, obj := range objects {
			if !isTranscribable(obj.FileName) {
				slog.Debug("checkTranscribable", "fname", obj.FileName, "isTranscribable", false)
				continue
			}

			if _, found := alreadyTranscribed[obj.FileName]; found {
				slog.Debug("checkTranscribed", "fname", obj.FileName, "completed", true)
				continue
			}
			slog.Debug("checkTranscribed", "fname", obj.FileName, "completed", false)

			initiateTranscribeJob(obj)
		}
	},
}

func init() {
	rootCmd.AddCommand(queryCmd)

	queryCmd.Flags().StringVarP(&bucketName, "bucket", "b", "streamboxdineorb", "S3 bucket name (required)")
	queryCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file for JSON data")
}

func listObjects(ctx context.Context, bucket, prefix string) ([]ObjectInfo, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-west-2"))
	if err != nil {
		return nil, fmt.Errorf("error loading AWS SDK config: %w", err)
	}

	client := s3.NewFromConfig(cfg)

	var objects []ObjectInfo

	input := &s3.ListObjectsV2Input{
		Bucket: &bucket,
		Prefix: &prefix,
	}

	paginator := s3.NewListObjectsV2Paginator(client, input)

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("error listing objects: %w", err)
		}

		for _, obj := range page.Contents {
			objectPath := fmt.Sprintf("s3://%s/%s", bucket, *obj.Key)
			fileName := *obj.Key
			fileSize := obj.Size
			modifiedDate := *obj.LastModified

			info := ObjectInfo{
				S3Bucket:     bucket,
				S3Path:       objectPath,
				FileName:     fileName,
				FileSize:     *fileSize,
				ModifiedDate: modifiedDate,
			}

			objects = append(objects, info)
		}
	}

	return objects, nil
}
