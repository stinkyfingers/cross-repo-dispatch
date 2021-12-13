const core = require('@actions/core');
const github = require('@actions/github');
const http = require('http');

try {
  const owner = core.getInput('owner');
  const repo = core.getInput('repo');
  const pat = core.getInput('pat');
  const sha = core.getInput('sha');

  const req = http.request({
    hostname: 'api.github.com',
    path: `/repos/${owner}/${repo}/actions/runs`,
    method: 'GET',

  }, res => {
    res.on('data', d => {
      core.setOutput('response', d);
    })
  }).end();

  req.on('error', error => {
    throw new Error(error);
  });

  core.setOutput('status', 'SUCCESSING');
  const payload = JSON.stringify(github.context.payload, undefined, 2)
  console.log(`The event payload: ${payload}`);
} catch (error) {
  core.setFailed(error.message);
}