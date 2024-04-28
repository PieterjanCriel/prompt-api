import { GoFunction } from '@aws-cdk/aws-lambda-go-alpha';
import * as cdk from 'aws-cdk-lib';
import { Duration } from 'aws-cdk-lib';
import { LambdaIntegration, RestApi } from 'aws-cdk-lib/aws-apigateway';
import { AttributeType, BillingMode, Table } from 'aws-cdk-lib/aws-dynamodb';
import { Role, ServicePrincipal } from 'aws-cdk-lib/aws-iam';
import { Runtime } from 'aws-cdk-lib/aws-lambda';
import { RetentionDays } from 'aws-cdk-lib/aws-logs';
import { Construct } from 'constructs';

export class PromptApiStack extends cdk.Stack {
  constructor(scope: Construct, id: string, props?: cdk.StackProps) {
    super(scope, id, props);

    const table = new Table(this, 'Prompts', {
      partitionKey: { name: 'reference', type: AttributeType.STRING },
      sortKey: { name: 'versioning', type: AttributeType.STRING },
      billingMode: BillingMode.PAY_PER_REQUEST,
    });

    const role = new Role(this, 'LambdaRole', {
      assumedBy: new ServicePrincipal('lambda.amazonaws.com'),
    });

    table.grantReadWriteData(role);
    
    const promptApiFunction = new GoFunction(this, 'handler', {
      entry: 'lambda/cmd/api',
      role: role,
      environment: {
        PROMPT_TABLE_NAME: table.tableName,
      },
      memorySize: 1024,
      timeout: Duration.seconds(2),
      runtime: Runtime.PROVIDED_AL2,
      logRetention: RetentionDays.ONE_WEEK,
    });
    
    const api = new RestApi(this, 'prompts-api', {
      restApiName: 'Prompts Service',
      description: 'This service serves prompts.',
    });

    const getIntegration = new LambdaIntegration(promptApiFunction);

    const prompt = api.root.addResource('prompt');

    prompt.addProxy({
      anyMethod: true,
      defaultIntegration: getIntegration,
    });
    
  }
}
