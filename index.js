const core = require('@actions/core');
const github = require('@actions/github');
const https = require('http');

try {
  const owner = core.getInput('owner');
  const repo = core.getInput('repo');
  const pat = core.getInput('pat');
  const sha = core.getInput('sha');

  const req = https.request({
    hostname: 'api.github.com',
    path: `/repos/${owner}/${repo}/actions/runs`,
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
      'accept': 'application/vnd.github.everest-preview+json',
      'Authorization': 'Basic ' + new Buffer.from(owner + ':' + pat).toString('base64')
    }
  }, res => {
    res.on('data', d => {
      core.setOutput('response', d);
    })
  });

  req.on('error', error => {
    console.log(error)
    core.setFailed(error.Message);
  });

  req.end();

  core.setOutput('status', 'SUCCESSING');
  // const payload = JSON.stringify(github.context.payload, undefined, 2)
  // console.log(`The event payload: ${payload}`);
} catch (error) {
  core.setFailed(error.message);
}