package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/brpaz/echozap"
	echo "github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/twilio/twilio-go"
	openapi "github.com/twilio/twilio-go/rest/api/v2010"
	"go.uber.org/zap"
)

type Response struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Response
	Error string `json:"error"`
}

func buildErrorResponse(code int, message string, err error) ErrorResponse {
	return ErrorResponse{
		Response: Response{
			Status:  code,
			Message: message,
		},
		Error: err.Error(),
	}
}

type MessageRequest struct {
	To      string `json:"to"`
	Body    string `json:"body"`
	Subject string `json:"subject,omitempty"`
}

type Secrets struct {
	TwilioSID               string
	TwilioToken             string
	TwilioPhoneNumber       string
	TwilioTargetPhoneNumber string
	SendgridFromUsername    string
	SendgridAPIKey          string
	SendgridFromEmail       string
}

func (s *Secrets) Validate() error {
	// fix this when twilio is ready for validation
	//	if s.TwilioSID == "" {
	//		return fmt.Errorf("secret not populated: TwilioSID")
	//	} else if s.TwilioToken == "" {
	//		return fmt.Errorf("secret not populated: TwilioToken")
	//	} else if s.TwilioPhoneNumber == "" {
	//		return fmt.Errorf("secret not populated: TwilioPhoneNumber")
	//	} else if s.TwilioTargetPhoneNumber == "" {
	//		return fmt.Errorf("secret not populated: TwilioTargetPhoneNumber")
	if s.SendgridAPIKey == "" {
		return fmt.Errorf("secret not populated: SendgridAPIKey")
	} else if s.SendgridFromEmail == "" {
		return fmt.Errorf("secret not populated: SendgridFromEmail")
	} else if s.SendgridFromUsername == "" {
		return fmt.Errorf("secret not populated: SendgridFromUsername")
	}
	return nil
}

func (s *Secrets) TwilioParams() twilio.ClientParams {
	return twilio.ClientParams{
		Username: s.TwilioSID,
		Password: s.TwilioToken,
	}
}

func (s *Secrets) TwilioMessageParams(message string) *openapi.CreateMessageParams {
	p := &openapi.CreateMessageParams{}
	p.SetTo(s.TwilioTargetPhoneNumber)
	p.SetFrom(s.TwilioPhoneNumber)
	p.SetBody(message)
	return p
}

func (s *Secrets) SendgridChunk(recipient string, subject string, body string) (*sendgrid.Client, *mail.SGMailV3) {
	from := mail.NewEmail(s.SendgridFromUsername, s.SendgridFromEmail)
	to := mail.NewEmail("Recipient", recipient)
	message := mail.NewSingleEmail(from, subject, to, body, body)
	client := sendgrid.NewSendClient(s.SendgridAPIKey)
	return client, message
}

func getSecrets() (*Secrets, error) {
	s := &Secrets{
		TwilioSID:               os.Getenv("TWILIO_ACCOUNT_SID"),
		TwilioToken:             os.Getenv("TWILIO_AUTH_TOKEN"),
		TwilioPhoneNumber:       os.Getenv("TWILIO_PHONE_NUMBER"),
		TwilioTargetPhoneNumber: os.Getenv("TO_PHONE_NUMBER"),
		SendgridFromUsername:    os.Getenv("SENDGRID_FROM_USERNAME"),
		SendgridAPIKey:          os.Getenv("SENDGRID_API_KEY"),
		SendgridFromEmail:       os.Getenv("SENDGRID_FROM_EMAIL"),
	}
	return s, s.Validate()
}

type Server struct {
	logger *zap.SugaredLogger
	echo   *echo.Echo
}

func (s *Server) Register() {
	s.echo = echo.New()
	s.echo.Use(middleware.RequestID())
	s.echo.Use(echozap.ZapLogger(s.logger.Desugar()))
	s.echo.GET("/health", s.handleHealth)
	sendGroup := s.echo.Group("/send")
	sendGroup.POST("/sms", s.handleSMS)
	sendGroup.GET("/linkedin", nil)
	sendGroup.POST("/email", s.handleEmail)
}

func (s *Server) Run() error {
	return s.echo.Start(":8080")
}

func (s *Server) handleHealth(c echo.Context) error {
	resp := &Response{
		Status:  200,
		Message: "API is running",
	}
	return c.JSON(resp.Status, resp)
}

func (s *Server) handleSMS(c echo.Context) error {
	req := &MessageRequest{}
	if err := json.NewDecoder(c.Request().Body).Decode(req); err != nil {

		// move this to middleware to reject empty erquests

		s.logger.Errorw("received malformed request", "body", c.Request().Body, "error", err)
		resp := buildErrorResponse(400, "Malformed request body", err)
		return c.JSON(resp.Status, resp)
	}

	sec, err := getSecrets()
	if err != nil {
		// likely forgot to set env var
		// make this not invisible
		s.logger.Errorw("failed to acquire secrets.", "error", err)
	}

	client := twilio.NewRestClientWithParams(sec.TwilioParams())
	p := sec.TwilioMessageParams("Hello from Golang!")

	m, err := client.Api.CreateMessage(p)
	if err != nil {
		s.logger.Errorw("failed to send message", "error", err)
		return c.JSON(500, buildErrorResponse(500, "error occurred communicating with twilio", err))
	}

	s.logger.Infow("successfully sent message", "price", m.Price, "message_status", m.Status)

	resp := &Response{
		Status:  200,
		Message: "Successfully sent!",
	}

	return c.JSON(200, resp)
}

func (s *Server) handleEmail(c echo.Context) error {
	req := &MessageRequest{}
	if err := json.NewDecoder(c.Request().Body).Decode(req); err != nil {
		s.logger.Errorw("received malformed request", "body", c.Request().Body, "error", err)
		resp := buildErrorResponse(400, "Malformed request body", err)
		return c.JSON(resp.Status, resp)
	}

	sec, err := getSecrets()
	if err != nil {
		// avoid screwing up envars
		s.logger.Errorw("failed to acquire secrets.", "error", err)
	}

	client, mail := sec.SendgridChunk(req.To, req.Subject, req.Body)

	m, err := client.Send(mail)
	if err != nil {
		s.logger.Errorw("failed to send message", "error", err, "headers", m.Headers, "body", m.Body)
		return c.JSON(500, buildErrorResponse(500, "error occurred sending email", err))
	}
	if m.StatusCode != 202 {
		s.logger.Errorw("failed to have message accepted", "code", m.StatusCode, "headers", m.Headers, "body", m.Body)
	}

	s.logger.Infow("successfully sent message", "status", m.StatusCode, "body", m.Body, "headers", m.Headers)

	resp := &Response{
		Status:  200,
		Message: "Successfully sent!",
	}

	return c.JSON(200, resp)
}

func main() {
	l, err := zap.NewProduction()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	logger := l.Sugar().Named("api")

	s := &Server{
		logger: logger,
	}

	s.Register()
	logger.Fatal(s.Run())
	return
}
