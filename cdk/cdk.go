package main

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsdynamodb"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambdaeventsources"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3"
	"github.com/aws/aws-cdk-go/awscdklambdagoalpha/v2"

	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

const functionDir = "../function"

type TextractInvoiceProcessingStackProps struct {
	awscdk.StackProps
}

func NewTextractInvoiceProcessingGolangStack(scope constructs.Construct, id string, props *TextractInvoiceProcessingStackProps) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	bucket := awss3.NewBucket(stack, jsii.String("invoice-images-input-bucket"), &awss3.BucketProps{
		BlockPublicAccess: awss3.BlockPublicAccess_BLOCK_ALL(),
		RemovalPolicy:     awscdk.RemovalPolicy_DESTROY,
		AutoDeleteObjects: jsii.Bool(true),
	})

	table := awsdynamodb.NewTable(stack, jsii.String("invoice-output-table"),
		&awsdynamodb.TableProps{
			PartitionKey: &awsdynamodb.Attribute{
				Name: jsii.String("source_file"),
				Type: awsdynamodb.AttributeType_STRING},

			TableName: jsii.String(*bucket.BucketName() + "_invoice_output"),
		})

	table.ApplyRemovalPolicy(awscdk.RemovalPolicy_DESTROY)

	function := awscdklambdagoalpha.NewGoFunction(stack, jsii.String("textract-function"),
		&awscdklambdagoalpha.GoFunctionProps{
			Runtime:     awslambda.Runtime_GO_1_X(),
			Environment: &map[string]*string{"TABLE_NAME": table.TableName()},
			Entry:       jsii.String(functionDir),
		})

	table.GrantWriteData(function)
	bucket.GrantRead(function, "*")
	function.Role().AddManagedPolicy(awsiam.ManagedPolicy_FromAwsManagedPolicyName(jsii.String("AmazonTextractFullAccess")))

	function.AddEventSource(awslambdaeventsources.NewS3EventSource(bucket, &awslambdaeventsources.S3EventSourceProps{
		Events: &[]awss3.EventType{awss3.EventType_OBJECT_CREATED},
	}))

	awscdk.NewCfnOutput(stack, jsii.String("invoice-input-bucket-name"),
		&awscdk.CfnOutputProps{
			ExportName: jsii.String("invoice-input-bucket-name"),
			Value:      bucket.BucketName()})

	awscdk.NewCfnOutput(stack, jsii.String("invoice-output-table-name"),
		&awscdk.CfnOutputProps{
			ExportName: jsii.String("invoice-output-table-name"),
			Value:      table.TableName()})

	return stack
}

func main() {
	app := awscdk.NewApp(nil)

	NewTextractInvoiceProcessingGolangStack(app, "TextractInvoiceProcessingGolangStack", &TextractInvoiceProcessingStackProps{
		awscdk.StackProps{
			Env: env(),
		},
	})

	app.Synth(nil)
}

func env() *awscdk.Environment {
	return nil
}
