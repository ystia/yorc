package main

import (
	"context"
	"time"

	"github.com/chromedp/chromedp"
)

func noHeadless(a *chromedp.ExecAllocator) {
	chromedp.Flag("headless", false)(a)
	// Like in Puppeteer.
	chromedp.Flag("hide-scrollbars", false)(a)
	chromedp.Flag("mute-audio", false)(a)
}

func (c *suiteContext) getChromeDPContext() (context.Context, context.CancelFunc) {
	var cancel context.CancelFunc
	var ctx context.Context

	if c.config.ChromeDP.Timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), time.Minute)
	}

	opts := chromedp.DefaultExecAllocatorOptions[:]
	if c.config.ChromeDP.NoHeadless {
		opts = append(opts, noHeadless)
	}

	allocCtx, c2 := chromedp.NewExecAllocator(ctx, opts...)
	if cancel == nil {
		cancel = c2
	}
	ctx, _ = chromedp.NewContext(allocCtx)
	return ctx, cancel
}
