package utils

import (
	"fmt"
	"strings"
)

// BuildPrompt menggabungkan session, history, retrieval, dan input baru menjadi prompt LLM.
func BuildPrompt(
	sessionID, userID, authStatus, currentIntent, lastBookingID, summary, keywords, lastMessage, history string,
	isWhatsapp bool,
	retrieval []string,
	newInput string,
	adminRole string,
	userName, userPhone, userEmail string,
	appInventory []map[string]interface{},
) string {

	// Format session info
	sessionInfoLines := []string{
		fmt.Sprintf("Session ID           : %s", sessionID),
		fmt.Sprintf("User ID              : %s", userID),
		fmt.Sprintf("Auth Status          : %s", authStatus),
		fmt.Sprintf("WhatsApp Mode        : %v", isWhatsapp),
		fmt.Sprintf("Current Intent       : %s", currentIntent),
		fmt.Sprintf("Last Booking ID      : %s", lastBookingID),
		fmt.Sprintf("Summary              : %s", summary),
		fmt.Sprintf("Keywords             : %s", keywords),
		fmt.Sprintf("Last Message         : %s", lastMessage),
	}

	if adminRole != "" {
		sessionInfoLines = append(sessionInfoLines, fmt.Sprintf("Admin Dashboard Role: %s", adminRole))
	}

	if userName != "" || userPhone != "" || userEmail != "" {
		sessionInfoLines = append(sessionInfoLines, fmt.Sprintf("User Registered Name  : %s", userName))
		sessionInfoLines = append(sessionInfoLines, fmt.Sprintf("User Registered Phone : %s", userPhone))
		sessionInfoLines = append(sessionInfoLines, fmt.Sprintf("User Registered Email : %s", userEmail))
	}

	sessionInfo := strings.Join(sessionInfoLines, "\n")

	// Format app inventory
	var appInventoryBlock string
	if len(appInventory) > 0 {
		var appLines []string
		for _, app := range appInventory {
			label, _ := app["label"].(string)
			pkgName, _ := app["package_name"].(string)
			if pkgName == "" {
				pkgName, _ = app["packageName"].(string)
			}
			isSys, _ := app["is_system"].(bool)
			if !isSys {
				isSys, _ = app["isSystem"].(bool)
			}
			
			sysStr := ""
			if isSys {
				sysStr = " (System App)"
			}
			appLines = append(appLines, fmt.Sprintf("  - %s: %s%s", label, pkgName, sysStr))
		}
		appInventoryBlock = strings.Join(appLines, "\n")
	} else {
		appInventoryBlock = "  (No apps synced or inventory is empty)"
	}

	// Format retrieval block
	retrievalBlock := "(tidak ada data tambahan)"
	if len(retrieval) > 0 {
		retrievalBlock = strings.Join(retrieval, "\n")
	}

	// Build final prompt
	prompt := fmt.Sprintf(`=== Session Info ===
%s

=== Installed Apps on Android Device ===
%s

=== Conversation History ===
%s

=== Retrieved Knowledge ===
%s

=== New User Input ===
User: %s`, sessionInfo, appInventoryBlock, history, retrievalBlock, newInput)

	return prompt
}
