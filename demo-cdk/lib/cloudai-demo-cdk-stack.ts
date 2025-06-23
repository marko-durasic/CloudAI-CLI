import * as cdk from 'aws-cdk-lib';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import * as apigateway from 'aws-cdk-lib/aws-apigateway';

export class CloudaiDemoCdkStack extends cdk.Stack {
  constructor(scope: cdk.App, id: string, props?: cdk.StackProps) {
    super(scope, id, props);

    const demoLambda = new lambda.Function(this, 'DemoLambda', {
      functionName: 'cloudai-demo-hello',
      runtime: lambda.Runtime.NODEJS_18_X,
      handler: 'index.handler',
      code: lambda.Code.fromInline(`
        exports.handler = async (event) => {
          return {
            statusCode: 200,
            body: JSON.stringify({ message: "Hello from Lambda!" })
          };
        };
      `),
    });

    const api = new apigateway.RestApi(this, 'DemoApi', {
      restApiName: 'cloudai-demo-api',
    });

    const hello = api.root.addResource('hello');
    hello.addMethod('GET', new apigateway.LambdaIntegration(demoLambda));
  }
} 