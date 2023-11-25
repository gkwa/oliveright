package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/transcribe"
	"github.com/aws/aws-sdk-go-v2/service/transcribe/types"
)

var (
	re               = regexp.MustCompile("[^A-Za-z0-9_.]")
	shouldTranscribe = map[string]types.MediaFormat{
		".mp3": types.MediaFormatMp3,
		".mp4": types.MediaFormatMp4,
		".m4a": types.MediaFormatM4a,
	}
)

func initiateTranscribeJob(s3Obj ObjectInfo) {
	fname := s3Obj.FileName

	if !isTranscribable(fname) {
		slog.Debug("checkTranscribable", "fname", fname, "skipping", true)
		return
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-west-2"))
	if err != nil {
		fmt.Println("Error loading AWS config:", err)
		os.Exit(1)
	}

	client := transcribe.NewFromConfig(cfg)

	bucketName := s3Obj.S3Bucket

	jobName := re.ReplaceAllString(fname, "_")

	languageCode := types.LanguageCodeEnUs
	mediaFormat := getMediaFormatFromExtension(fname)
	mediaFileUri := fmt.Sprintf("s3://%s/%s", bucketName, fname)
	yourOutputBucket := bucketName

	input := &transcribe.StartTranscriptionJobInput{
		TranscriptionJobName: &jobName,
		LanguageCode:         languageCode,
		MediaFormat:          mediaFormat,
		Media: &types.Media{
			MediaFileUri: &mediaFileUri,
		},
		OutputBucketName: &yourOutputBucket,
	}
	slog.Debug("initiateTranscribeJob", "fname", fname, "jobName", jobName)

	jobInput := transcribe.GetTranscriptionJobInput{
		TranscriptionJobName: &jobName,
	}

	client.GetTranscriptionJob(context.TODO(), &jobInput)

	output, err := client.GetTranscriptionJob(context.TODO(), &jobInput)
	if err != nil {
		fmt.Println("Error:", err)
	}

	if err == nil {
		switch output.TranscriptionJob.TranscriptionJobStatus {
		case types.TranscriptionJobStatusCompleted:
			slog.Debug("initiateTranscribeJob", "fname", fname, "jobname", jobName, "status", output.TranscriptionJob.TranscriptionJobStatus)
			return
		case types.TranscriptionJobStatusFailed:
			if output.TranscriptionJob.FailureReason != nil {
				slog.Error("initiateTranscribeJob", "fname", fname, "jobname", jobName, "status", output.TranscriptionJob.TranscriptionJobStatus, "failureReason", *output.TranscriptionJob.FailureReason)
			} else {
				slog.Error("initiateTranscribeJob", "fname", fname, "jobname", jobName, "status", output.TranscriptionJob.TranscriptionJobStatus)
			}
			return
		default:
			fmt.Println("Transcription Job is in another status:", output.TranscriptionJob.TranscriptionJobStatus)
		}
	}

	resp, err := client.StartTranscriptionJob(context.TODO(), input)
	if err != nil {
		fmt.Println("Error starting transcription job:", err)
		os.Exit(1)
	}

	fmt.Println("Transcription Job ID:", *resp.TranscriptionJob.TranscriptionJobName)
}

func getMediaFormatFromExtension(fname string) types.MediaFormat {
	extension := strings.ToLower(filepath.Ext(fname))
	return shouldTranscribe[extension]
}

func isTranscribable(filename string) bool {
	extension := strings.ToLower(filepath.Ext(filename))
	_, found := shouldTranscribe[extension]
	return found
}

func genTranscriptionCompletedMap(objectInfos []ObjectInfo) map[string]ObjectInfo {
	fileNameMap := make(map[string]ObjectInfo)

	for _, objInfo := range objectInfos {
		fileNameMap[objInfo.FileName] = objInfo
	}

	completed := make(map[string]ObjectInfo)

	for fileName, objInfo := range fileNameMap {
		if strings.ToLower(filepath.Ext(fileName)) == ".json" {
			continue
		}

		s := re.ReplaceAllString(fmt.Sprintf("%s.json", fileName), "_")
		if _, found := fileNameMap[s]; found {
			completed[fileName] = objInfo
		}
	}

	return completed
}
