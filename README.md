# OllamaChat

## Introduction

ollamachat is a Go application that demonstrates how to interact with a Local Language Model (LLM) server. This application showcases handling streaming responses efficiently using Go's concurrency model and manages request cancellations using the `context` package. The user interface is built with the Fyne framework.

## Acknowledgments

Shout out to [Fyne](https://fyne.io/) for providing a powerful and intuitive GUI library that simplifies our development process. This project is made possible by the contributions of the Fyne community, and their efforts in creating an excellent and promising framework is appreciated.

## Setup Instructions

1. **Install Go**: Ensure you have Go installed. You can download it from [the official website](https://golang.org/dl/).
2. **Build the Project**: Navigate to the project directory in your terminal and execute `go build` to compile the code and produce an executable.
3. **Run the Program**: Use `./ollamachat` to start the application.

## Usage

1. **Launch the Program**: Execute the binary without additional arguments.
2. **Fetch Available Models**: Upon startup, the program retrieves a list of models from the LLM server.
3. **Select a Model**: Users can select a model from a dropdown to determine which model the LLM server will use for query processing.
4. **Submit a Query**: Type a query into the text field and send it to the LLM.
5. **Receive Streaming Response**: The application processes the LLM's streaming responses and updates the UI in real-time.

## Concurrency

For efficient execution and a responsive UI, the application leverages Go's concurrency features:

- **Goroutines for Long Operations**: Queries to the LLM are sent using goroutines. This prevents the main UI thread from blocking.

  ```go
  go c.sendMessageToLLM(ctx, selectedModel, query, scrollContainer)
  ```

- **Context for Request Cancellation**: The context package is utilized to allow for user-initiated request cancelation.

  ```go
  ctx, cancelFunc := context.WithCancel(context.Background())
  ```

## Streaming Responses

The application handles streaming responses from the LLM to deliver real-time interactions:

- **Incremental JSON Decoding**: The `json.Decoder` is used to parse JSON responses incrementally as they're received, allowing the UI to update dynamically.

  ```go
  decoder := json.NewDecoder(resp.Body)
  ```

- **Response Handling Loop**: Continuously processes incoming JSON objects to update the chat UI with the LLM's responses.

  ```go
  for {
      var llmResp map[string]interface{}
      if err := decoder.Decode(&llmResp); errors.Is(err, io.EOF) {
          break
      } else if err != nil {
          return err
      }

      if responseText, ok := llmResp["response"].(string); ok {
          updateChat(responseText, newStream)
          newStream = false
      }
  }
  ```

## Context Handling

Utilize the context package to manage request life cycles efficiently:

- **Cancellation**: Users can cancel ongoing requests through a cancel button in the UI.

  ```go
  cancelButton.OnTapped = func() {
      if c.cancelFunc != nil {
          c.cancelFunc()
          addMessageCard("\n\n**Request canceled**", false)
      }
  }
  ```

## User Interface

### Drop-Down Width Configuration

For model selection, the drop-down widget dynamically sizes to fit the longest model name:

- **Determine Longest Model Name**: The app iterates through model names to find the longest one.

  ```go
  longestModel := ""
  for _, model := range c.modelSelect.Options {
      if len(model) > len(longestModel) {
          longestModel = model
      }
  }
  ```

- **Set Minimum Width**: The width of the drop-down is set using the size of the longest model name plus extra padding for better readability.

  ```go
  textSize := canvas.NewText(longestModel, nil).MinSize()
  modelSelectContainer := container.NewGridWrap(fyne.NewSize(textSize.Width+50, c.modelSelect.MinSize().Height), c.modelSelect)
  ```

  The `+50` padding ensures the drop-down accommodates extra UI elements while avoiding cramped text, enhancing user interface aesthetics.

## Troubleshooting

- **LLM Not Responding**: Consider increasing the context timeout.
- **Invalid JSON Response**: Verify LLM logs to diagnose errors in the response.

## References

- "The Go Programming Language" by Brian Kernighan and Al Aho
- "HTTP in Go" by Miek Gieben and Jelle van der Velden
- "Context package documentation" by Rob Pike

## License

This project is released under the MIT License. Please refer to the `LICENSE` file for more details.

