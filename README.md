# Ghost 👻
Ghost is an experimental CLI that intelligently scaffolds a GitHub Action workflow based on your local application stack, using OpenAI.

Using file names and natural language, Ghost generates a GitHub Action skeleton.

## Getting started
1. First, you'll need to set up an [OpenAI API key](https://platform.openai.com/account/api-keys).
2. Run `ghost config set OPENAI_API_KEY <your key here>` with your key from step 1
3. Run `ghost run` to start project analysis of the current working directory.

