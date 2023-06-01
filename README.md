# ghost

A CLI ghost that intelligently scaffolds a GitHub Action workflow based on your local application stack, using OpenAI.

- no .github/workflow dir?
    - ghost runs on the cwd
    - prints the summary of the ghas it'll use
    - user can course correct
    - when accepted, file is spit out into the .github/workflow dir
    - gives you a summary of next steps

- already have a .github/workflow dir?
    - ghost confirms you want to update the workflow and asks what you'd like to add/change/remove
    - prints the summary of the ghas it'll use
    - user can course correct
    - when accepted, file is spit out into the .github/workflow dir
    - gives you a summary of next steps


- vscode extension
    - surface gesture via command palette?