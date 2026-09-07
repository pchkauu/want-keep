package processor

import (
	"bytes"
	"context"
	"image"
	_ "image/jpeg"
	"image/png"

	application "github.com/pchkauu/want-keep/backend/internal/attachments/application"
	domain "github.com/pchkauu/want-keep/backend/internal/attachments/domain"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

func (e *Engine) image(ctx context.Context, media string, data []byte) (application.Inspection, error) {
	if err := ctx.Err(); err != nil {
		return application.Inspection{Reason: domain.LimitExceeded}, nil
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return application.Inspection{Reason: domain.InvalidDocument}, nil
	}
	if "image/"+format != media {
		return application.Inspection{Reason: domain.UnsupportedContent}, nil
	}
	if config.Width < 1 || config.Height < 1 || config.Width > 40_000_000/config.Height {
		return application.Inspection{Reason: domain.LimitExceeded}, nil
	}
	source, format, err := image.Decode(bytes.NewReader(data))
	if err != nil || "image/"+format != media {
		return application.Inspection{Reason: domain.InvalidDocument}, nil
	}
	width, height := config.Width, config.Height
	if width > 2048 || height > 2048 {
		if width >= height {
			height = max(1, height*2048/width)
			width = 2048
		} else {
			width = max(1, width*2048/height)
			height = 2048
		}
	}
	preview := image.NewNRGBA(image.Rect(0, 0, width, height))
	draw.ApproxBiLinear.Scale(preview, preview.Bounds(), source, source.Bounds(), draw.Src, nil)
	var encoded bytes.Buffer
	if err = png.Encode(&encoded, preview); err != nil {
		return application.Inspection{}, domain.ErrUnavailable
	}
	if err = ctx.Err(); err != nil {
		return application.Inspection{Reason: domain.LimitExceeded}, nil
	}
	return application.Inspection{Pages: []application.Preview{{Width: width, Height: height, PNG: encoded.Bytes()}}}, nil
}
