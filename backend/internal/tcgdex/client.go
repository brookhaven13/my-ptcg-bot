package tcgdex

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const baseURL = "https://api.tcgdex.net/v2/zh-tw"

type Client struct {
	http *http.Client
}

func NewClient() *Client {
	return &Client{
		http: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) GetCard(setID, localID string) (*TCGdexCard, error) {
	url := fmt.Sprintf("%s/sets/%s/%s", baseURL, setID, localID)

	resp, err := c.http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch card %s/%s: %w", setID, localID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("card not found: %s/%s", setID, localID)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d for %s/%s", resp.StatusCode, setID, localID)
	}

	var card TCGdexCard
	if err := json.NewDecoder(resp.Body).Decode(&card); err != nil {
		return nil, fmt.Errorf("decode card %s/%s: %w", setID, localID, err)
	}

	return &card, nil
}

func (c *Client) GetCardRaw(setID, localID string) (*TCGdexCard, []byte, error) {
	url := fmt.Sprintf("%s/sets/%s/%s", baseURL, setID, localID)

	resp, err := c.http.Get(url)
	if err != nil {
		return nil, nil, fmt.Errorf("fetch card %s/%s: %w", setID, localID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil, fmt.Errorf("card not found: %s/%s", setID, localID)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("unexpected status %d for %s/%s", resp.StatusCode, setID, localID)
	}

	var raw json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, nil, fmt.Errorf("decode raw %s/%s: %w", setID, localID, err)
	}

	var card TCGdexCard
	if err := json.Unmarshal(raw, &card); err != nil {
		return nil, nil, fmt.Errorf("decode card %s/%s: %w", setID, localID, err)
	}

	return &card, raw, nil
}

// SearchCardByName 用中文名搜尋 TCGdex 卡片，回傳第一筆結果及原始 JSON
func (c *Client) SearchCardByName(name string) (*TCGdexCard, []byte, error) {
	searchURL := fmt.Sprintf("%s/cards?name=%s", baseURL, url.QueryEscape(name))

	resp, err := c.http.Get(searchURL)
	if err != nil {
		return nil, nil, fmt.Errorf("search card %q: %w", name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("search %q: status %d", name, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("read search response: %w", err)
	}

	var results []TCGdexCard
	if err := json.Unmarshal(body, &results); err != nil {
		// 可能回傳單一物件而非陣列
		var single TCGdexCard
		if err2 := json.Unmarshal(body, &single); err2 != nil {
			return nil, nil, fmt.Errorf("decode search results for %q: %w", name, err)
		}
		raw, _ := json.Marshal(single)
		return &single, raw, nil
	}

	if len(results) == 0 {
		return nil, nil, nil
	}

	first := results[0]
	setID := first.Set.ID
	localID := first.LocalID

	// 搜尋結果通常不含 set 物件，從 id 解析（如 "SV6-100" → set="SV6", local="100"）
	if setID == "" && localID != "" && strings.HasSuffix(first.ID, "-"+localID) {
		setID = first.ID[:len(first.ID)-len(localID)-1]
	}

	if setID != "" && localID != "" {
		card, raw, err := c.GetCardRaw(setID, localID)
		if err == nil {
			return card, raw, nil
		}
	}

	raw, _ := json.Marshal(first)
	return &first, raw, nil
}
