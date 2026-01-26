package nmap

import (
	"testing"
)

func TestParseClassLine(t *testing.T) {
	// 模拟一个包含 Class 行的 nmap-os-db 内容
	dbContent := `
Fingerprint Microsoft Windows 10
Class Microsoft | Windows | 10 | general purpose
CPE cpe:/o:microsoft:windows_10
SEQ(SP=105-10F%GCD=1-6%ISR=108-112%TI=I%CI=I%II=I%SS=S%TS=U)
OPS(O1=M5B4NW8NNS%O2=M5B4NW8NNS%O3=M5B4NW8%O4=M5B4NW8NNS%O5=M5B4NW8NNS%O6=M5B4NNS)
WIN(W1=2000%W2=2000%W3=2000%W4=2000%W5=2000%W6=2000)
ECN(R=Y%DF=Y%T=80%W=2000%O=M5B4NW8NNS%CC=N%Q=)
T1(R=Y%DF=Y%T=80%S=O%A=S+%F=AS%RD=0%Q=)
T2(R=Y%DF=Y%T=80%W=0%S=Z%A=S%F=AR%O=%RD=0%Q=)
T3(R=Y%DF=Y%T=80%W=0%S=Z%A=O%F=AR%O=%RD=0%Q=)
T4(R=Y%DF=Y%T=80%W=0%S=A%A=O%F=R%O=%RD=0%Q=)
T5(R=Y%DF=Y%T=80%W=0%S=Z%A=S+%F=AR%O=%RD=0%Q=)
T6(R=Y%DF=Y%T=80%W=0%S=A%A=O%F=R%O=%RD=0%Q=)
T7(R=Y%DF=Y%T=80%W=0%S=Z%A=S+%F=AR%O=%RD=0%Q=)
U1(R=Y%DF=N%T=80%IPL=164%UN=0%RIPL=G%RID=G%RIPCK=G%RUCK=G%RUD=G)
IE(R=Y%DFI=N%T=80%CD=Z)
`

	db, err := ParseOSDB(dbContent)
	if err != nil {
		t.Fatalf("Failed to parse OS DB: %v", err)
	}

	if len(db.Fingerprints) != 1 {
		t.Fatalf("Expected 1 fingerprint, got %d", len(db.Fingerprints))
	}

	fp := db.Fingerprints[0]

	// 验证字段解析
	if fp.Vendor != "Microsoft" {
		t.Errorf("Expected Vendor 'Microsoft', got '%s'", fp.Vendor)
	}
	if fp.OSFamily != "Windows" {
		t.Errorf("Expected OSFamily 'Windows', got '%s'", fp.OSFamily)
	}
	if fp.OSGen != "10" {
		t.Errorf("Expected OSGen '10', got '%s'", fp.OSGen)
	}
	if fp.Device != "general purpose" {
		t.Errorf("Expected Device 'general purpose', got '%s'", fp.Device)
	}

	// 验证 RawFingerprint (String() 方法)
	raw := fp.String()
	if raw == "" {
		t.Error("Expected non-empty raw fingerprint string")
	}
}
