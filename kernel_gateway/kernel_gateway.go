package kernel_gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"github.com/satori/go.uuid"
)

type KernelHeader struct {
	UserName string `json:"username"`
	Version  string `json:"version"`
	Session  string `json:"session"`
	MsgID    string `json:"msg_id"`
	MsgType  string `json:"msg_type"`
}
type ParentHeader struct{}
type ExecuteContent struct {
	Code            string      `json:"code"`
	Silent          bool        `json:"silent"`
	StoreHistroy    bool        `json:"store_history"`
	UserExpressions interface{} `json:"store_expressions"`
	AllowSTDIN      bool        `json:"allow_stdin"`
}
type Metadata struct{}
type Buffers struct{}
type ExecuteMessage struct {
	Header       KernelHeader   `json:"header"`
	ParentHeader ParentHeader   `json:"parent_header"`
	Channel      string         `json:"channel"`
	Content      ExecuteContent `json:"content"`
	Metdata      Metadata       `json:"metadata"`
	Buffers      Buffers        `json:"buffers"`
}

const (
	APIEndpointBase = "http://localhost:8888"
	WSEndpointBase  = "ws://localhost:8888"
)

type Client struct {
	apiBaseURL *url.URL
	wsBaseURL  *url.URL
	httpClient *http.Client
	wsClient   *websocket.Conn
	Kernel     *Kernel
}

func NewClient() (*Client, error) {
	// 初期化
	apiBaseURL, _ := url.Parse(APIEndpointBase)
	wsBaseURL, _ := url.Parse(WSEndpointBase)
	c := &Client{
		httpClient: http.DefaultClient,
		apiBaseURL: apiBaseURL,
		wsBaseURL:  wsBaseURL,
	}
	ctx := context.Background()

	// サーバに接続してKernelを作成
	kernel, err := c.NewKernel(ctx, "")
	if err != nil {
		return nil, err
	}
	c.Kernel = kernel
	fmt.Println("Server Connected")
	fmt.Printf("Kernel ID: %s\n", c.Kernel.ID)

	// WebSocket接続
	wsUrlStr := fmt.Sprintf("/api/kernels/%s/channels", c.Kernel.ID)
	rel, err := url.Parse(wsUrlStr)
	if err != nil {
		return nil, err
	}
	wsUrl := c.wsBaseURL.ResolveReference(rel)

	conn, _, err := websocket.DefaultDialer.Dial(wsUrl.String(), nil)
	if err != nil {
		return nil, err
	}
	c.wsClient = conn
	fmt.Println("Connected WebSocket")
	defer conn.Close()

	done := make(chan struct{})

	go func() {
		defer conn.Close()
		defer close(done)

		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println(string(message))
		}
	}()

	// message送信
	c.NewWSWrite(&ExecuteMessage{
		Header: KernelHeader{
			Version: "5.0",
			MsgID:   uuid.NewV4().String(),
			MsgType: "execute_request",
		},
		ParentHeader: struct{}{},
		Channel:      "shell",
		Content: ExecuteContent{
			Code:            "print(\"hoge\")",
			Silent:          false,
			StoreHistroy:    false,
			UserExpressions: struct{}{},
			AllowSTDIN:      false,
		},
		Metdata: struct{}{},
		Buffers: struct{}{},
	})

	time.Sleep(5 * time.Second)

	return c, nil
}

func (c *Client) NewWSWrite(body interface{}) error {
	var buf bytes.Buffer
	if body != nil {
		err := json.NewEncoder(&buf).Encode(body)
		if err != nil {
			return err
		}
	}

	if err := c.wsClient.WriteMessage(websocket.TextMessage, buf.Bytes()); err != nil {
		return err
	}

	return nil
}

func (c *Client) NewRequest(method, urlStr string, body interface{}) (*http.Request, error) {
	rel, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	u := c.apiBaseURL.ResolveReference(rel)

	var buf io.ReadWriter
	if body != nil {
		buf = new(bytes.Buffer)
		err := json.NewEncoder(buf).Encode(body)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, u.String(), buf)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}

func (c *Client) Do(ctx context.Context, req *http.Request, v interface{}) (*http.Response, error) {
	req = req.WithContext(ctx)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() {
		io.CopyN(ioutil.Discard, resp.Body, 512)
		resp.Body.Close()
	}()

	if v != nil {
		if w, ok := v.(io.Writer); ok {
			io.Copy(w, resp.Body)
		} else {
			err = json.NewDecoder(resp.Body).Decode(v)
			if err == io.EOF {
				err = nil
			}
		}
	}

	fmt.Print(resp)

	return resp, err
}
