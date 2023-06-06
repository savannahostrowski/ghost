# Ghost 👻
Ghost is an experimental CLI that intelligently scaffolds a GitHub Action workflow based on your local application stack, using OpenAI.

Using file names and natural language, Ghost generates a GitHub Action skeleton.

## Getting started
1. First, you'll need to set up an [OpenAI API key](https://platform.openai.com/account/api-keys).
2. Set the API key in your environment variables as `OPENAI_API_KEY` (e.g. `export OPENAI_API_KEY=<your-key-here>`)
3. Run `ghost run` to start project analysis of the current working directory.


## Upcoming features
- Ability to set other models (currently uses GPT 3.5 Turbo) using a config file
- VS Code extension
