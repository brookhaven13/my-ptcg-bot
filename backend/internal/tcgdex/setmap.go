package tcgdex

import "fmt"

// 已驗證的系列基底 ID（從 asia.pokemon-card.com/tw 確認）
var setImageBaseID = map[string]int{
	// Scarlet & Violet 系列（TCGdex 系列碼）
	"SV6":  10415,
	"SV6a": 10583,
	"SV7":  10896,
	"SV7a": 11031,
	"SV8":  11181,
	"SV8a": 11526,
	"SV9":  12463,
	"SV9a": 12659,
	"SV10": 12751,

	// TW 官網擴充標記（牌組匯入用）
	"MC":  16472, // 大師球合集（742 張）
	"M2":  14319, // 大師球 2（80 張）
	"M2a": 14661, // 大師球 2a（193 張，注意：部分 ID 不連續）
	"MTL": 19284, // 大師球訓練家聯盟（23 張）
	"MTK": 19353, // 大師球訓練家對決（23 張）

	// 基本能量
	"SVE": 10149,
}

// GetImageURL 根據系列碼和卡號計算官方 TW 網站圖片 URL
func GetImageURL(setID string, localID int) string {
	baseID, ok := setImageBaseID[setID]
	if !ok {
		return ""
	}
	return fmt.Sprintf("https://asia.pokemon-card.com/tw/card-img/tw%08d.png", baseID+localID-1)
}
