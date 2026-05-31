package agents

import (
	"context"
	"encoding/json"
	"fmt"

	"ecomteam/internal/llm"
)

// QualityCheck (QA reviewer) validates the finished listing.
type QualityCheck struct{}

func (QualityCheck) Name() string { return NameQC }

func (QualityCheck) Run(ctx context.Context, d *StageData, client llm.Client, p Progress) error {
	const task = "Checking product data"
	p(NameQC, 40, task)

	// Deterministic structural checks first.
	if len(d.Listing.SellingPoints) == 0 || d.Listing.Headline == "" || len(d.ImagePNG) == 0 {
		d.Listing.QCStatus = "needs_fix"
		d.Listing.QCNotes = "ข้อมูลไม่ครบ: ต้องมีจุดขาย พาดหัว และรูปภาพ"
		p(NameQC, 100, "Needs fix")
		return nil
	}

	system := "agent:qc\n" +
		"You are a QA reviewer for marketplace listings. Check that the headline, promotion and " +
		"selling points are consistent and that nothing looks misleading. " + langInstruction(d.Lang) +
		` Respond as JSON: {"qc_status":"passed"|"needs_fix","qc_notes":"..."}`
	user := fmt.Sprintf("Headline: %s\nPromotion: %s\nSelling points: %v",
		d.Listing.Headline, d.Listing.Promotion, d.Listing.SellingPoints)

	out, err := client.Chat(ctx, system, user)
	if err != nil {
		// QC is non-fatal: default to passed with a note rather than failing the job.
		d.Listing.QCStatus = "passed"
		d.Listing.QCNotes = "ตรวจอัตโนมัติไม่สำเร็จ ผ่านโดยใช้การตรวจโครงสร้างพื้นฐาน"
		p(NameQC, 100, "Done")
		return nil
	}
	var parsed struct {
		QCStatus string `json:"qc_status"`
		QCNotes  string `json:"qc_notes"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil || parsed.QCStatus == "" {
		parsed.QCStatus = "passed"
		parsed.QCNotes = "ตรวจผ่านการตรวจโครงสร้างพื้นฐาน"
	}
	d.Listing.QCStatus = parsed.QCStatus
	d.Listing.QCNotes = parsed.QCNotes
	p(NameQC, 100, "Done")
	return nil
}
