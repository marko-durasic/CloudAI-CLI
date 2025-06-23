#!/usr/bin/env node
import * as cdk from 'aws-cdk-lib';
import { CloudaiDemoCdkStack } from '../lib/cloudai-demo-cdk-stack';

const app = new cdk.App();
new CloudaiDemoCdkStack(app, 'CloudaiDemoCdkStack'); 