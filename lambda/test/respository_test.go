// File: repository_test.go

package respository

import (
	"api/pkg/prompt"
	"api/pkg/repository"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockDynamoDBAPI is a mock of the DynamoDBAPI interface for testing.
type mockDynamoDBAPI struct {
	mock.Mock
	dynamodbiface.DynamoDBAPI
}

func (m *mockDynamoDBAPI) Query(input *dynamodb.QueryInput) (*dynamodb.QueryOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*dynamodb.QueryOutput), args.Error(1)
}

func (m *mockDynamoDBAPI) PutItem(input *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*dynamodb.PutItemOutput), args.Error(1)
}

func (m *mockDynamoDBAPI) TransactWriteItems(input *dynamodb.TransactWriteItemsInput) (*dynamodb.TransactWriteItemsOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*dynamodb.TransactWriteItemsOutput), args.Error(1)
}

func TestGetPrompt(t *testing.T) {
	// Setting up the environment variable
	os.Setenv("PROMPT_TABLE_NAME", "Prompts")

	// Creating instances and setting expectations
	db := new(mockDynamoDBAPI)
	repo := repository.NewRepository(db)
	expectedPrompt := &prompt.Prompt{Namespace: "test", Name: "example", Version: "1.2.3"}

	db.On("Query", mock.AnythingOfType("*dynamodb.QueryInput")).Return(&dynamodb.QueryOutput{
		Items: []map[string]*dynamodb.AttributeValue{
			{
				"reference":  {S: aws.String("test#example")},
				"versioning": {S: aws.String("0001.0002.0003")},
				"Namespace":  {S: aws.String("test")},
				"Name":       {S: aws.String("example")},
				"Version":    {S: aws.String("0001.0002.0003")},
			},
		},
	}, nil)

	result, err := repo.GetPrompt("test#example", "1.2.3")
	assert.NoError(t, err)
	assert.Equal(t, expectedPrompt, result)
}

func TestPutPrompt(t *testing.T) {
	// Setting up the environment variable
	os.Setenv("PROMPT_TABLE_NAME", "Prompts")

	// Creating instances and setting expectations
	db := new(mockDynamoDBAPI)
	repo := repository.NewRepository(db)
	inputPrompt := &prompt.Prompt{Namespace: "test", Name: "example", Version: "1.2.3"}

	db.On("TransactWriteItems", mock.AnythingOfType("*dynamodb.TransactWriteItemsInput")).Return(&dynamodb.TransactWriteItemsOutput{}, nil)

	err := repo.PutPrompt(inputPrompt)
	assert.NoError(t, err)
}
