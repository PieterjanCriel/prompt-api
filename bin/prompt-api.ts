#!/usr/bin/env node
import 'source-map-support/register';
import * as cdk from 'aws-cdk-lib';
import { PromptApiStack } from '../lib/prompt-api-stack';

const app = new cdk.App();
new PromptApiStack(app, 'PromptApiStack', {
  env: { account: process.env.CDK_DEFAULT_ACCOUNT, region: process.env.CDK_DEFAULT_REGION },
});