package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/textract"
	"github.com/aws/aws-sdk-go-v2/service/textract/types"
)

var textractClient *textract.Client
var dynamodbClient *dynamodb.Client
var table string

func init() {
	table = os.Getenv("TABLE_NAME")

	if table == "" {
		log.Fatal("missing environment variable TABLE_NAME")
	}

	cfg, err := config.LoadDefaultConfig(context.Background())

	if err != nil {
		log.Fatal("failed to load config ", err)
	}

	textractClient = textract.NewFromConfig(cfg)
	dynamodbClient = dynamodb.NewFromConfig(cfg)

}

func handler(ctx context.Context, s3Event events.S3Event) {
	for _, record := range s3Event.Records {

		fmt.Println("file", record.S3.Object.Key, "uploaded to", record.S3.Bucket.Name)

		sourceBucketName := record.S3.Bucket.Name
		fileName := record.S3.Object.Key

		err := invoiceProcessing(sourceBucketName, fileName)

		if err != nil {
			log.Fatal("failed to process file ", record.S3.Object.Key, " in bucket ", record.S3.Bucket.Name, err)
		}
	}
}

func main() {
	lambda.Start(handler)
}

func invoiceProcessing(sourceBucketName, fileName string) error {

	resp, err := textractClient.AnalyzeExpense(context.Background(), &textract.AnalyzeExpenseInput{
		Document: &types.Document{
			S3Object: &types.S3Object{
				Bucket: &sourceBucketName,
				Name:   &fileName,
			},
		},
	})

	if err != nil {
		return err
	}

	for _, doc := range resp.ExpenseDocuments {
		item := make(map[string]ddbTypes.AttributeValue)

		item["source_file"] = &ddbTypes.AttributeValueMemberS{Value: fileName}

		for _, summaryField := range doc.SummaryFields {

			if *summaryField.Type.Text == "INVOICE_RECEIPT_ID" {
				fmt.Println(*summaryField.Type.Text, "=", *summaryField.ValueDetection.Text)
				item["receipt_id"] = &ddbTypes.AttributeValueMemberS{Value: *summaryField.ValueDetection.Text}
			} else if *summaryField.Type.Text == "TOTAL" {
				fmt.Println(*summaryField.Type.Text, "=", *summaryField.ValueDetection.Text)
				item["total"] = &ddbTypes.AttributeValueMemberS{Value: *summaryField.ValueDetection.Text}
			} else if *summaryField.Type.Text == "INVOICE_RECEIPT_DATE" {
				fmt.Println(*summaryField.Type.Text, "=", *summaryField.ValueDetection.Text)
				item["receipt_date"] = &ddbTypes.AttributeValueMemberS{Value: *summaryField.ValueDetection.Text}
			} else if *summaryField.Type.Text == "DUE_DATE" {
				fmt.Println(*summaryField.Type.Text, "=", *summaryField.ValueDetection.Text)
				item["due_date"] = &ddbTypes.AttributeValueMemberS{Value: *summaryField.ValueDetection.Text}
			}
		}

		_, err := dynamodbClient.PutItem(context.Background(), &dynamodb.PutItemInput{
			TableName: aws.String(table),
			Item:      item,
		})

		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("invoice item added to table")
	}

	return nil
}
