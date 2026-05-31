package llm

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"strings"
)

// Mock is a deterministic Client used when AI_MODE=mock. It inspects an agent
// tag embedded in the system prompt (see agents package) and returns canned,
// valid JSON, plus a generated placeholder PNG for images.
type Mock struct{}

// NewMock returns a deterministic mock client.
func NewMock() *Mock { return &Mock{} }

// Chat returns canned JSON appropriate to the calling agent.
func (m *Mock) Chat(_ context.Context, system, user string) (string, error) {
	switch {
	case strings.Contains(system, "agent:benefit"):
		return `{"selling_points":["ใช้งานได้จริง คุ้มราคา","วัสดุคุณภาพดี ทนทาน","ส่งไว มีรับประกัน"]}`, nil
	case strings.Contains(system, "agent:promo"):
		return `{"headline":"ของดีต้องมีติดบ้าน!","promotion":"ซื้อ 1 แถม 1 ส่งฟรีทั่วประเทศ"}`, nil
	case strings.Contains(system, "agent:design"):
		return `{"layout":"สินค้าอยู่กลางภาพ ข้อความพาดหัวด้านบน ป้ายโปรฯ มุมขวาล่าง","color_tone":"ส้ม-ขาว สดใส สไตล์ Shopee"}`, nil
	case strings.Contains(system, "agent:prompt"):
		return `{"image_prompt":"A professional e-commerce product listing image. Bright orange and white studio background. Bold Thai headline text at the top. A red promo badge in the lower-right corner. Clean commercial product photography, high quality, sharp lighting."}`, nil
	case strings.Contains(system, "agent:qc"):
		return `{"qc_status":"passed","qc_notes":"ข้อความอ่านง่าย ราคาและโปรโมชันสอดคล้องกัน รูปครบถ้วน"}`, nil
	case strings.Contains(system, "agent:aff_profile"):
		return `{"bio":"นักการตลาดออนไลน์สายช่วยแม่ค้าปั้นร้านให้ขายดี ถนัดคอนเทนต์รีวิวสินค้าและไลฟ์ขายของ","niche":"แม่ค้าออนไลน์ / SME","pitch":"ช่วยร้านค้าทำรูปสินค้าระดับมือโปรด้วย AI เพิ่มยอดขายแบบไม่ต้องจ้างกราฟิก"}`, nil
	case strings.Contains(system, "agent:aff_content"):
		return `{"posts":["🔥 ลงของขายแล้วไม่มีคนซื้อ? ให้ AI ทำรูปสินค้าสวยๆ พร้อมป้ายโปรฯ ในพริบตา ลองเลย!","✨ ทีม AI 6 ตัวช่วยคิดจุดขาย เขียนแคปชั่น ทำรูป จบในที่เดียว ร้านไหนยังไม่ใช้ถือว่าพลาด!","🛒 อยากให้ลูกค้ากดใส่ตะกร้ารัวๆ? เริ่มฟรีได้เลยวันนี้"]}`, nil
	case strings.Contains(system, "agent:aff_promote"):
		return `{"headline":"ตัวนี้คุ้มมาก บอกเลย!","caption":"🔥 ใครกำลังมองหาอยู่ ตัวนี้รีวิวดีมาก ของแท้ ส่งไว ราคาคุ้มสุดๆ กดเลยก่อนของหมด! 🛒✨","hashtags":["#ของดีบอกต่อ","#รีวิวสินค้า","#Shopeeหาของถูก","#ลดราคา","#คุ้มเกินราคา"]}`, nil
	case strings.Contains(system, "agent:aff_reco"):
		return `{"recommendations":[{"category":"ความงาม/สกินแคร์","reason":"ซื้อซ้ำสูง คอนเทนต์รีวิวทำง่าย"},{"category":"แฟชั่น/เครื่องแต่งกาย","reason":"ภาพสินค้าสวยดึงดูด คอนเวอร์ชันดี"},{"category":"ของใช้ในบ้าน","reason":"กลุ่มลูกค้ากว้าง ตัดสินใจซื้อไว"},{"category":"แกตเจ็ต/อุปกรณ์ไอที","reason":"ราคาต่อชิ้นสูง คอมมิชชันคุ้ม"}]}`, nil
	default:
		return `{}`, nil
	}
}

// Image returns a generated placeholder PNG (no external service needed).
func (m *Mock) Image(_ context.Context, prompt string) ([]byte, error) {
	const w, h = 512, 512
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	// Diagonal two-tone "Shopee" orange/white blocks so the placeholder is
	// visually distinct and deterministic.
	orange := color.RGBA{R: 0xEE, G: 0x4D, B: 0x2D, A: 0xFF}
	cream := color.RGBA{R: 0xFF, G: 0xF3, B: 0xE6, A: 0xFF}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if ((x/32)+(y/32))%2 == 0 {
				img.Set(x, y, orange)
			} else {
				img.Set(x, y, cream)
			}
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
