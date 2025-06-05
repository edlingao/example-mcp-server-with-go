package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/edlingao/hexago/internal/calculator/core"
	"github.com/edlingao/hexago/internal/calculator/ports/driven"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type CalculatorMCPAdapter struct {
	CalculatorService driven.CalculatorOperations
	DBService         driven.StoringOperations[core.Calculation]
	MCPServer         *server.MCPServer
}

func NewCalculatorMCPAdapter(
	calcService driven.CalculatorOperations,
	dbService driven.StoringOperations[core.Calculation],
	mcpServer *server.MCPServer,
) CalculatorMCPAdapter {
	mcpAdapter := CalculatorMCPAdapter{
		CalculatorService: calcService,
		DBService:         dbService,
		MCPServer:         mcpServer,
	}

	calcutatorTool := mcp.NewTool("calculate",
		mcp.WithDescription("Perform a calculation based on the provided operation"),
		mcp.WithNumber("operation",
			mcp.Required(),
			mcp.Description(
				"The operation to perform: 0 for addition, 1 for subtraction, 2 for multiplication, 3 for division",
			),
		),
		mcp.WithNumber("num1",
			mcp.Required(),
			mcp.Description("The first number for the calculation"),
		),
		mcp.WithNumber("num2",
			mcp.Required(),
			mcp.Description("The second number for the calculation"),
		),
	)

	historyResourceTool := mcp.NewTool("history",
		mcp.WithDescription("Retrieve the history of calculations performed by the calculator service"),
		mcp.WithDescription("Use 'All' to retrieve all calculations or specify an ID to retrieve a specific calculation"),
		mcp.WithDescription("To retrieve all calculations, use 'All' as the ID"),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("The ID of the calculation history to retrieve"),
		),
	)

	historyResourceTemplate := mcp.NewResourceTemplate(
		"calculation://{id}",
		"Calculator History",
		mcp.WithTemplateDescription("A resource containing the history of calculations performed by the calculator service"),
		mcp.WithTemplateMIMEType("application/json"),
	)

	mcpAdapter.MCPServer.AddTool(calcutatorTool, mcpAdapter.Calculate)
	mcpAdapter.MCPServer.AddTool(historyResourceTool, mcpAdapter.HistoryTool)
	mcpAdapter.MCPServer.AddResourceTemplate(historyResourceTemplate, mcpAdapter.History)

	return mcpAdapter
}

func (m *CalculatorMCPAdapter) Calculate(c context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	operation, err := request.RequireInt("operation")
	if err != nil {
		log.Printf("Error retrieving operation: %v", err)
		return nil, err
	}

	num1, err := request.RequireInt("num1")
	if err != nil {
		log.Printf("Error retrieving num1: %v", err)
		return nil, err
	}

	num2, err := request.RequireInt("num2")
	if err != nil {
		log.Printf("Error retrieving num2: %v", err)
		return nil, err
	}

	result := m.CalculatorService.Calculate(num1, num2, operation)

	return mcp.NewToolResultText(fmt.Sprintf("The result of %d %s %d is %d", num1, m.CalculatorService.GetSymbol(operation), num2, result)), nil
}

func (m *CalculatorMCPAdapter) History(c context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	id, err := m.GetIDFromURI(request.Params.URI)
	if err != nil {
		log.Printf("Error retrieving table from URI: %v", err)
		return nil, err
	}

	log.Print("Fetching history for table: ", id)
	calculation, err := m.DBService.Get(id, "calculations")
	if err != nil {
		log.Printf("Error retrieving calculations from database: %v", err)
		return nil, err
	}

	jsonData, err := json.Marshal(calculation)
	if err != nil {
		log.Printf("Error marshalling calculations to JSON: %v", err)
		return nil, err
	}

	resources := []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      fmt.Sprintf("calculation://%s", id),
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}
	return resources, nil
}

func (m *CalculatorMCPAdapter) HistoryTool(c context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := request.RequireString("id")
	if err != nil {
		log.Printf("Error retrieving id: %v", err)
		return nil, err
	}

	if id == "All" {
		// If the id is "All", we return all calculations
		calculations := m.DBService.GetAll("calculations")
		jsonData, err := json.Marshal(calculations)
		if err != nil {
			log.Printf("Error marshalling calculations to JSON: %v", err)
			return nil, err
		}
		return mcp.NewToolResultText(string(jsonData)), nil
	}

	log.Print("Fetching history for id: ", id)
	calculation, err := m.DBService.Get(id, "calculations")
	if err != nil {
		log.Printf("Error retrieving calculations from database: %v", err)
		return nil, err
	}

	jsonData, err := json.Marshal(calculation)
	if err != nil {
		log.Printf("Error marshalling calculations to JSON: %v", err)
		return nil, err
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}

func (m *CalculatorMCPAdapter) GetIDFromURI(uri string) (string, error) {
	// Assuming the URI is in the format "sqlite3://database/history/{id}"
	// Extracting the ID from the URI
	parts := strings.Split(uri, "/")
	if len(parts) < 3 {
		return "", fmt.Errorf("invalid URI format: %s", uri)
	}
	return parts[2], nil
}
