package aic

type TokenCtx struct {
	reader *PromptReader
	index  int
}

func (c *TokenCtx) bind(r *PromptReader, index int) {
	c.reader = r
	c.index = index
}

func (c *TokenCtx) Reader() *PromptReader { return c.reader }
func (c *TokenCtx) Index() int            { return c.index }

func (c *TokenCtx) Prev() PromptToken {
	if c.reader == nil {
		return nil
	}
	i := c.index - 1
	if i < 0 || i >= len(c.reader.Tokens) {
		return nil
	}
	return c.reader.Tokens[i]
}

func (c *TokenCtx) Next() PromptToken {
	if c.reader == nil {
		return nil
	}
	i := c.index + 1
	if i < 0 || i >= len(c.reader.Tokens) {
		return nil
	}
	return c.reader.Tokens[i]
}
