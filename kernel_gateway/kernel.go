package kernel_gateway

import (
	"context"
)

type Kernel struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

func (c *Client) NewKernel(ctx context.Context, kernelId string) (*Kernel, error) {
	if len(kernelId) == 0 {
		req, err := c.NewRequest("POST", "/api/kernels", &struct {
			Name string `json:"name"`
		}{
			Name: "python3",
		})

		kResp := new(Kernel)
		_, err = c.Do(ctx, req, kResp)
		if err != nil {
			return nil, err
		}

		return kResp, nil
	} else {
		kernel := Kernel{
			Name: "python3",
			ID:   kernelId,
		}

		return &kernel, nil
	}
}
