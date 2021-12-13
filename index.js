const core = require('@actions/core');
const github = require('@actions/github');

try {
  const owner = core.getInput('owner');
  const repo = core.getInput('repo');
  const pat = core.getInput('pat');
  const sha = core.getInput('sha');

  core.setOuput('status', 'SUCCESSING');
  const payload = JSON.stringify(github.context.payload, undefined, 2)
  console.log(`The event payload: ${payload}`);
} catch (error) {
  core.setFailed(error.message);
}