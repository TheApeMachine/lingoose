package main

import (
	"context"
	"fmt"
	"os"

	openaiembedder "github.com/henomis/lingoose/embedder/openai"
	"github.com/henomis/lingoose/index"
	"github.com/henomis/lingoose/llm/openai"
	"github.com/henomis/lingoose/loader"
	"github.com/henomis/lingoose/prompt"
	"github.com/henomis/lingoose/textsplitter"
	pineconego "github.com/henomis/pinecone-go"
	pineconerequest "github.com/henomis/pinecone-go/request"
	pineconeresponse "github.com/henomis/pinecone-go/response"
)

func main() {

	openaiEmbedder, err := openaiembedder.New(openaiembedder.AdaEmbeddingV2)
	if err != nil {
		panic(err)
	}

	pineconeApiKey := os.Getenv("PINECONE_API_KEY")
	if pineconeApiKey == "" {
		panic("PINECONE_API_KEY is not set")
	}

	pineconeEnvironment := os.Getenv("PINECONE_ENVIRONMENT")
	if pineconeEnvironment == "" {
		panic("PINECONE_ENVIRONMENT is not set")
	}

	pineconeClient := pineconego.New(pineconeEnvironment, pineconeApiKey)

	whoamiReq := &pineconerequest.Whoami{}
	whoamiResp := &pineconeresponse.Whoami{}

	err = pineconeClient.Whoami(context.Background(), whoamiReq, whoamiResp)
	if err != nil {
		panic(err)
	}

	pineconeIndex, err := index.NewPinecone("test", whoamiResp.ProjectID, openaiEmbedder)
	if err != nil {
		panic(err)
	}

	indexSize, err := pineconeIndex.Size()
	if err != nil {
		panic(err)
	}

	if indexSize == 0 {
		loader, err := loader.NewDirectoryLoader(".", ".txt")
		if err != nil {
			panic(err)
		}

		documents, err := loader.Load()
		if err != nil {
			panic(err)
		}

		textSplitter := textsplitter.NewRecursiveCharacterTextSplitter(1000, 20, nil, nil)

		documentChunks := textSplitter.SplitDocuments(documents)

		for _, doc := range documentChunks {
			fmt.Println(doc.Content)
			fmt.Println("----------")
			fmt.Println(doc.Metadata)
			fmt.Println("----------")
			fmt.Println()

		}

		err = pineconeIndex.LoadFromDocuments(context.Background(), documentChunks)
		if err != nil {
			panic(err)
		}
	}

	query := "What is the purpose of the NATO Alliance?"
	topk := 3
	similarities, err := pineconeIndex.SimilaritySearch(
		context.Background(),
		query,
		&topk,
	)
	if err != nil {
		panic(err)
	}

	for _, similarity := range similarities {
		fmt.Printf("Similarity: %f\n", similarity.Score)
		fmt.Printf("Document: %s\n", similarity.Document.Content)
		fmt.Println("Metadata: ", similarity.Document.Metadata)
		fmt.Println("----------")
	}

	llmOpenAI, err := openai.New(openai.GPT3TextDavinci003, openai.DefaultOpenAITemperature, openai.DefaultOpenAIMaxTokens, true)
	if err != nil {
		panic(err)
	}

	prompt1, err := prompt.NewPromptTemplate(
		"Based on the following context answer to the question.\n\nContext:\n{{.context}}\n\nQuestion: {{.query}}",
		map[string]string{
			"query":   query,
			"context": similarities[0].Document.Content,
		},
	)
	if err != nil {
		panic(err)
	}

	err = prompt1.Format(nil)
	if err != nil {
		panic(err)
	}

	llmOpenAI.Completion(context.Background(), prompt1.Prompt())

}