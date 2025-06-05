package configurator

import (
	"log"
	"os"

	calculatorAdapter "github.com/edlingao/hexago/internal/calculator/adapters"
	calculatorCore "github.com/edlingao/hexago/internal/calculator/core"
	"github.com/mark3labs/mcp-go/server"

	_ "github.com/joho/godotenv/autoload"
	"github.com/labstack/echo/v4"
)

type Configurator struct {
	CalculatorHandler    calculatorAdapter.CalculatorHandler
	CalculatorWebPage    calculatorAdapter.CalculatorWebpage
	CalculatorMCPAdapter calculatorAdapter.CalculatorMCPAdapter
	Echo                 *echo.Echo
	v1                   *echo.Group
	root                 *echo.Group
}

func New(
	echo *echo.Echo,
) *Configurator {
	// V1
	api := echo.Group("/api")
	v1 := api.Group("/v1")

	root := echo.Group("")

	return &Configurator{
		Echo: echo,
		v1:   v1,
		root: root,
	}
}

func (c *Configurator) AddCalculatorAPI() *Configurator {
	dbService := calculatorAdapter.NewDB[calculatorCore.Calculation]()
	calcService := calculatorCore.NewCalculator(dbService)
	calculatorHttpService := c.v1.Group("/calculator")

	calculationHandler := calculatorAdapter.NewCalculatorHandler(
		"/calculator",
		calculatorHttpService,
		calcService,
		dbService,
	)

	c.CalculatorHandler = *calculationHandler

	return c
}

func (c *Configurator) AddCalculatorWeb() *Configurator {
	dbService := calculatorAdapter.NewDB[calculatorCore.Calculation]()
	calcService := calculatorCore.NewCalculator(dbService)
	calculatorWebpageService := calculatorAdapter.NewCalculatorWebpage(
		"/",
		c.root,
		calcService,
		dbService,
	)

	c.CalculatorWebPage = *calculatorWebpageService
	return c
}

func (c *Configurator) AddCalculatorMCP() *Configurator {
	dbService := calculatorAdapter.NewDB[calculatorCore.Calculation]()
	calcService := calculatorCore.NewCalculator(dbService)

	mcpServer := server.NewMCPServer(
		"Calculator MCP Server",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, true),
		server.WithPromptCapabilities(true),
		server.WithRecovery(),
	)

	calculatorMCPAdapter := calculatorAdapter.NewCalculatorMCPAdapter(
		calcService,
		dbService,
		mcpServer,
	)

	c.CalculatorMCPAdapter = calculatorMCPAdapter

	return c
}

func (c *Configurator) Start() {
	port := os.Getenv("GO_PORT")
	c.Echo.Logger.Fatal(c.Echo.Start(":" + port))
	log.Printf("Server started on port %s", port)
}

func (c *Configurator) StartMCP() {
	port := os.Getenv("GO_MCP_PORT")
	if port == "" {
		port = "5000" // Default port if not set
	}

	log.Printf("Starting MCP server on port %s", port)
	httpServer := server.NewSSEServer(c.CalculatorMCPAdapter.MCPServer)
	log.Fatal(httpServer.Start(":" + port))
}
