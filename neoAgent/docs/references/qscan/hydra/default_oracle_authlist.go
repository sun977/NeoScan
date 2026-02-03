package hydra

func DefaultOracleList() *AuthList {
	a := NewAuthList()
	a.Username = []string{
		"sys",
		"system",
		"admin",
		"test",
		"web",
		"orcl",
	}
	a.Password = []string{
		"123456", "admin", "admin123", "root", "", "pass123", "pass@123", "password", "Password", "P@ssword123", "123123", "654321", "111111", "123", "1", "admin@123", "Admin@123", "admin123!@#", "{user}", "{user}1", "{user}111", "{user}123", "{user}@123", "{user}_123", "{user}#123", "{user}@111", "{user}@2019", "{user}@123#4", "P@ssw0rd!", "P@ssw0rd", "Passw0rd", "qwe123", "12345678", "test", "test123", "123qwe", "123qwe!@#", "123456789", "123321", "666666", "a123456.", "123456~a", "123456!a", "000000", "1234567890", "8888888", "!QAZ2wsx", "1qaz2wsx", "abc123", "abc123456", "1qaz@WSX", "a11111", "a12345", "Aa1234", "Aa1234.", "Aa12345", "a123456", "a123123", "Aa123123", "Aa123456", "Aa12345.", "sysadmin", "system", "1qaz!QAZ", "2wsx@WSX", "qwe123!@#", "Aa123456!", "A123456s!", "sa123456", "1q2w3e", "Charge123", "Aa123456789", "elastic123",
	}
	a.Special = []Auth{
		NewSpecialAuth("internal", "oracle"),
		NewSpecialAuth("system", "manager"),
		NewSpecialAuth("system", "oracle"),
		NewSpecialAuth("sys", "change_on_install"),
		NewSpecialAuth("SYS", "CHANGE_ON_INSTALLorINTERNAL"),
		NewSpecialAuth("SYSTEM", "MANAGER"),
		NewSpecialAuth("OUTLN", "OUTLN"),
		NewSpecialAuth("SCOTT", "TIGER"),
		NewSpecialAuth("ADAMS", "WOOD"),
		NewSpecialAuth("JONES", "STEEL"),
		NewSpecialAuth("CLARK", "CLOTH"),
		NewSpecialAuth("BLAKE", "PAPER."),
		NewSpecialAuth("HR", "HR"),
		NewSpecialAuth("OE", "OE"),
		NewSpecialAuth("SH", "SH"),
		NewSpecialAuth("DBSNMP", "DBSNMP"),
		NewSpecialAuth("sysman", "oem_temp"),
		NewSpecialAuth("aqadm", "aqadm"),
		NewSpecialAuth("ANONYMOUS", "ANONYMOUS"),
		NewSpecialAuth("CTXSYS", "CTXSYS"),
		NewSpecialAuth("DIP", "DIP"),
		NewSpecialAuth("DMSYS", "DMSYS"),
		NewSpecialAuth("EXFSYS", "EXFSYS"),
		NewSpecialAuth("MDDATA", "MDDATA"),
		NewSpecialAuth("MDSYS", "MDSYS"),
		NewSpecialAuth("MGMT_VIEW", "MGMT_VIEW"),
		NewSpecialAuth("OLAPSYS", "MANAGER"),
		NewSpecialAuth("ORDPLUGINS", "ORDPLUGINS"),
		NewSpecialAuth("ORDSYS", "ORDSYS"),
		NewSpecialAuth("WK_TEST", "WK_TEXT"),
	}
	return a
}
