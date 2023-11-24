package cmd

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
)

var (
	bucketName string
	filter     string
	outputFile string
)

// ObjectInfo represents information about an S3 object
type ObjectInfo struct {
	S3Path       string
	FileName     string
	FileSize     int64
	ModifiedDate time.Time
}

// queryCmd represents the query command
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

		// Print the collected object information
		for _, obj := range objects {
			// Skip objects that don't match the filter
			if filter != "" && !matchesFilter(obj.FileName) {
				continue
			}

			fmt.Printf("S3 Path: %s\n", obj.S3Path)
			fmt.Printf("Filename: %s\n", obj.FileName)
			fileSizeString := humanize.Bytes(uint64(obj.FileSize))
			fmt.Printf("File Size: %s\n", fileSizeString)
			fmt.Printf("Modified Date: %s\n", obj.ModifiedDate.String())

			// Calculate and print the age of the file
			age := calculateAge(obj.ModifiedDate)
			fmt.Printf("Age: %s\n", age)

			fmt.Println("----------------------------")
		}
	},
}

func init() {
	rootCmd.AddCommand(queryCmd)

	// Local flags for specifying the S3 bucket name, filter, and output file
	queryCmd.Flags().StringVarP(&bucketName, "bucket", "b", "streamboxdineorb", "S3 bucket name (required)")
	queryCmd.Flags().StringVarP(&filter, "filter", "f", "", "Regex pattern for filtering filenames")
	queryCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file for JSON data")
}

func listObjects(ctx context.Context, bucket, prefix string) ([]ObjectInfo, error) {
	// Load AWS SDK configuration
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-west-2"))
	if err != nil {
		return nil, fmt.Errorf("error loading AWS SDK config: %w", err)
	}

	// Create an S3 client
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

			// Create an instance of the ObjectInfo struct
			info := ObjectInfo{
				S3Path:       objectPath,
				FileName:     fileName,
				FileSize:     *fileSize,
				ModifiedDate: modifiedDate,
			}

			// Append the information to the slice
			objects = append(objects, info)
		}
	}

	return objects, nil
}

// calculateAge calculates the age of a file based on the modified date
func calculateAge(modifiedDate time.Time) string {
	return humanize.RelTime(modifiedDate, time.Now(), "ago", "from now")
}

// matchesFilter checks if the filename matches the filter regex
func matchesFilter(filename string) bool {
	if filter == "" {
		return true
	}
	regex, err := regexp.Compile(filter)
	if err != nil {
		fmt.Println("Error compiling filter regex:", err)
		return false
	}
	return regex.MatchString(filename)
}
