"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || (function () {
    var ownKeys = function(o) {
        ownKeys = Object.getOwnPropertyNames || function (o) {
            var ar = [];
            for (var k in o) if (Object.prototype.hasOwnProperty.call(o, k)) ar[ar.length] = k;
            return ar;
        };
        return ownKeys(o);
    };
    return function (mod) {
        if (mod && mod.__esModule) return mod;
        var result = {};
        if (mod != null) for (var k = ownKeys(mod), i = 0; i < k.length; i++) if (k[i] !== "default") __createBinding(result, mod, k[i]);
        __setModuleDefault(result, mod);
        return result;
    };
})();
Object.defineProperty(exports, "__esModule", { value: true });
exports.ArchTrainingStack = void 0;
const cdk = __importStar(require("aws-cdk-lib"));
const lambda = __importStar(require("aws-cdk-lib/aws-lambda"));
const s3 = __importStar(require("aws-cdk-lib/aws-s3"));
const iam = __importStar(require("aws-cdk-lib/aws-iam"));
const sfn = __importStar(require("aws-cdk-lib/aws-stepfunctions"));
const tasks = __importStar(require("aws-cdk-lib/aws-stepfunctions-tasks"));
const events = __importStar(require("aws-cdk-lib/aws-events"));
const targets = __importStar(require("aws-cdk-lib/aws-events-targets"));
const ec2 = __importStar(require("aws-cdk-lib/aws-ec2"));
/**
 * ArchTrainingStack sets up:
 * 1. S3 bucket to store raw metadata & training artefacts
 * 2. Lambda to collect AWS architecture metadata → S3 (raw/)
 * 3. Lambda to convert raw JSON → JSONL friendly for fine-tuning (train/)
 * 4. Step-Functions State Machine that orchestrates:
 *      Collect → Prep → SageMaker TrainingJob
 * 5. EventBridge rule that triggers the State Machine on a cron schedule
 */
class ArchTrainingStack extends cdk.Stack {
    constructor(scope, id, props = {}) {
        super(scope, id, props);
        const bucket = new s3.Bucket(this, 'ArchTrainingBucket', {
            encryption: s3.BucketEncryption.S3_MANAGED,
            lifecycleRules: [
                { expiration: cdk.Duration.days(30) }, // keep artefacts 30 days
            ],
        });
        //---------------------------------------------------------------------
        // 1. Data-collection Lambda
        //---------------------------------------------------------------------
        const collectorRole = new iam.Role(this, 'CollectorRole', {
            assumedBy: new iam.ServicePrincipal('lambda.amazonaws.com'),
        });
        // Basic execution & read-only permissions
        collectorRole.addManagedPolicy(iam.ManagedPolicy.fromAwsManagedPolicyName('service-role/AWSLambdaBasicExecutionRole'));
        collectorRole.addManagedPolicy(iam.ManagedPolicy.fromAwsManagedPolicyName('ReadOnlyAccess'));
        bucket.grantWrite(collectorRole);
        const dataCollector = new lambda.Function(this, 'DataCollectorFn', {
            runtime: lambda.Runtime.PYTHON_3_11,
            handler: 'index.handler',
            code: lambda.Code.fromInline(`import boto3, json, os, datetime, tempfile, gzip
import urllib.parse

def handler(event, context):
    """Collect high-level AWS resource metadata and upload to S3 as gzipped JSON."""
    session = boto3.Session()
    resources = []

    # Lambda functions example – extend with more AWS services
    lambda_client = session.client('lambda')
    for page in lambda_client.get_paginator('list_functions').paginate():
        for fn in page.get('Functions', []):
            resources.append({'Type': 'Lambda', 'FunctionName': fn['FunctionName'], 'Arn': fn['FunctionArn']})

    # TODO: add API Gateway, SNS, SQS, etc.

    # Dump to tmp file then upload
    ts = datetime.datetime.utcnow().strftime('%Y%m%dT%H%M%SZ')
    key = f"raw/{ts}.json.gz"
    tmp = tempfile.NamedTemporaryFile(delete=False)
    with gzip.open(tmp.name, 'wt', encoding='utf-8') as f:
        json.dump(resources, f)
    s3 = session.client('s3')
    bucket = os.environ['BUCKET_NAME']
    s3.upload_file(tmp.name, bucket, key)
    return {'status': 'ok', 'objects': len(resources), 's3_key': key}
`),
            role: collectorRole,
            timeout: cdk.Duration.minutes(5),
            environment: {
                BUCKET_NAME: bucket.bucketName,
            },
        });
        //---------------------------------------------------------------------
        // 2. Data-prep Lambda (JSON → JSONL For fine-tune)
        //---------------------------------------------------------------------
        const prepRole = new iam.Role(this, 'PrepRole', {
            assumedBy: new iam.ServicePrincipal('lambda.amazonaws.com'),
        });
        prepRole.addManagedPolicy(iam.ManagedPolicy.fromAwsManagedPolicyName('service-role/AWSLambdaBasicExecutionRole'));
        bucket.grantReadWrite(prepRole);
        const dataPrep = new lambda.Function(this, 'DataPrepFn', {
            runtime: lambda.Runtime.PYTHON_3_11,
            handler: 'index.handler',
            code: lambda.Code.fromInline(`import boto3, json, os, gzip, io, datetime, tempfile

def handler(event, context):
    """Convert collected JSON to JSONL pairs for training (simple demo)."""
    s3 = boto3.client('s3')
    bucket = os.environ['BUCKET_NAME']
    key = event.get('raw_key')
    if not key:
        raise ValueError('raw_key missing')

    obj = s3.get_object(Bucket=bucket, Key=key)
    raw = json.loads(gzip.decompress(obj['Body'].read()).decode('utf-8'))

    # Very naive Q&A pair generation
    jsonl_lines = []
    for r in raw:
        if r['Type'] == 'Lambda':
            q = f"What triggers the {r['FunctionName']} Lambda?"
            a = f"It is triggered by X (placeholder – user needs richer context)."
            jsonl_lines.append(json.dumps({'prompt': q, 'completion': a}))

    ts = datetime.datetime.utcnow().strftime('%Y%m%dT%H%M%SZ')
    out_key = f"train/{ts}.jsonl"
    s3.put_object(Bucket=bucket, Key=out_key, Body='\n'.join(jsonl_lines).encode('utf-8'))
    return {'status': 'ok', 'train_key': out_key, 'records': len(jsonl_lines)}
`),
            role: prepRole,
            timeout: cdk.Duration.minutes(5),
            environment: {
                BUCKET_NAME: bucket.bucketName,
            },
        });
        //---------------------------------------------------------------------
        // 3. SageMaker training task definition
        //---------------------------------------------------------------------
        // IAM role that SageMaker uses inside the training container
        const sagemakerExecRole = new iam.Role(this, 'SageMakerExecRole', {
            assumedBy: new iam.ServicePrincipal('sagemaker.amazonaws.com'),
        });
        bucket.grantReadWrite(sagemakerExecRole);
        sagemakerExecRole.addManagedPolicy(iam.ManagedPolicy.fromAwsManagedPolicyName('AmazonSageMakerFullAccess'));
        const trainingJobTask = new tasks.SageMakerCreateTrainingJob(this, 'CreateTrainingJob', {
            trainingJobName: sfn.JsonPath.stringAt('$$.Execution.Name'),
            algorithmSpecification: {
                algorithmName: 'xgboost', // Placeholder built-in algorithm, replace with LLM fine-tune image as needed
                trainingInputMode: tasks.InputMode.FILE,
            },
            inputDataConfig: [
                {
                    channelName: 'training',
                    dataSource: {
                        s3DataSource: {
                            s3DataType: tasks.S3DataType.S3_PREFIX,
                            s3Location: tasks.S3Location.fromBucket(bucket, 'train/'),
                        },
                    },
                    contentType: 'application/jsonlines',
                },
            ],
            outputDataConfig: {
                s3OutputLocation: tasks.S3Location.fromBucket(bucket, 'model-artifacts'),
            },
            resourceConfig: {
                instanceCount: 1,
                instanceType: new ec2.InstanceType('ml.m5.large'),
                volumeSize: cdk.Size.gibibytes(30),
            },
            role: sagemakerExecRole,
            stoppingCondition: {
                maxRuntime: cdk.Duration.hours(2),
            },
            integrationPattern: sfn.IntegrationPattern.RUN_JOB,
        });
        //---------------------------------------------------------------------
        // 4. Assemble Step Functions workflow
        //---------------------------------------------------------------------
        const invokeCollector = new tasks.LambdaInvoke(this, 'InvokeCollector', {
            lambdaFunction: dataCollector,
            payloadResponseOnly: true,
            outputPath: '$',
        });
        const invokePrep = new tasks.LambdaInvoke(this, 'InvokePrep', {
            lambdaFunction: dataPrep,
            payload: sfn.TaskInput.fromObject({ raw_key: sfn.JsonPath.stringAt('$.s3_key') }),
            payloadResponseOnly: true,
            outputPath: '$',
        });
        const definition = invokeCollector.next(invokePrep).next(trainingJobTask);
        const stateMachine = new sfn.StateMachine(this, 'ArchTrainingStateMachine', {
            definition,
            timeout: cdk.Duration.hours(3),
        });
        //---------------------------------------------------------------------
        // 5. EventBridge schedule
        //---------------------------------------------------------------------
        const minutes = props.scheduleMinutes ?? 1440; // default daily
        new events.Rule(this, 'ArchTrainingSchedule', {
            schedule: events.Schedule.rate(cdk.Duration.minutes(minutes)),
            targets: [new targets.SfnStateMachine(stateMachine)],
        });
    }
}
exports.ArchTrainingStack = ArchTrainingStack;
