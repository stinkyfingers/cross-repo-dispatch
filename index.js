const core = require('@actions/core');
const github = require('@actions/github');
const fetch = require('node-fetch');


async function run() {
  try {
    const owner = core.getInput('owner');
    const repo = core.getInput('repo');
    const pat = core.getInput('pat');
    const sha = core.getInput('sha');

    const resp = await fetch(`https://api.github.com/repos/${owner}/${repo}/actions/runs`);
    const results = await resp.json();
    core.setOutput('results', results)

    core.setOutput('status', 'SUCCESSING');
    const payload = JSON.stringify(github.context.payload, undefined, 2)
    console.log(`The event payload: ${payload}`);
  } catch (error) {
    core.setFailed(error.message);
  }
}

run();