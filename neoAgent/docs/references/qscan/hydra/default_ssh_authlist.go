package hydra

func DefaultSshList() *AuthList {
	a := NewAuthList()
	a.Username = []string{
		"root",
		"admin",
		//"test",
		//"user",
		//"root",
		//"manager",
		//"webadmin",
	}
	a.Password = []string{
		"123456", "admin", "admin123", "root", "", "pass123", "pass@123", "password", "Password", "P@ssword123", "123123", "654321", "111111", "123", "1", "admin@123", "Admin@123", "admin123!@#", "{user}", "{user}1", "{user}111", "{user}123", "{user}@123", "{user}_123", "{user}#123", "{user}@111", "{user}@2019", "{user}@123#4", "P@ssw0rd!", "P@ssw0rd", "Passw0rd", "qwe123", "12345678", "test", "test123", "123qwe", "123qwe!@#", "123456789", "123321", "666666", "a123456.", "123456~a", "123456!a", "000000", "1234567890", "8888888", "!QAZ2wsx", "1qaz2wsx", "abc123", "abc123456", "1qaz@WSX", "a11111", "a12345", "Aa1234", "Aa1234.", "Aa12345", "a123456", "a123123", "Aa123123", "Aa123456", "Aa12345.", "sysadmin", "system", "1qaz!QAZ", "2wsx@WSX", "qwe123!@#", "Aa123456!", "A123456s!", "sa123456", "1q2w3e", "Charge123", "Aa123456789", "elastic123",
	}
	a.Special = []Auth{
		NewSpecialAuth("db2admin", "db2admin"),
		NewSpecialAuth("oracle", "oracle"),
		NewSpecialAuth("mysql", "mysql"),
		NewSpecialAuth("postgres", "postgres"),
		NewSpecialAuth("kali", "kali"),
		NewSpecialAuth("defaultUsername", "defaultPassword"),
		NewSpecialAuth("admin", "Admin@huawei"),
		NewSpecialAuth("admin", "admin@huawei.com"),
		NewSpecialAuth("admin", "admin"),
		NewSpecialAuth("admin", "Admin@123"),
		NewSpecialAuth("admin", "Changeme_123"),
		NewSpecialAuth("admin", "Admin123"),
		NewSpecialAuth("admin", "Changeme123"),
		NewSpecialAuth("admin", "Admin@storage"),
		NewSpecialAuth("admin", "eSight@123"),
		NewSpecialAuth("admin", "Cis#BigData123"),
		NewSpecialAuth("admin", "Huawei@123"),
		NewSpecialAuth("LogAdmin", "Changeme123"),
		NewSpecialAuth("SystemAdmin", "Changeme123"),
		NewSpecialAuth("OperateAdmin", "Changeme123"),
		NewSpecialAuth("administrator", "Changeme123"),
		NewSpecialAuth("Administrator", "Admin@9000"),
		NewSpecialAuth("Administrator", "Changeme@321"),
		NewSpecialAuth("api-admin", "admin@123"),
		NewSpecialAuth("root", "admin123"),
		NewSpecialAuth("root", "mduadmin"),
		NewSpecialAuth("root", "Changeme_123"),
		NewSpecialAuth("root", "admin"),
		NewSpecialAuth("root", "adminHW"),
		NewSpecialAuth("root", "password"),
		NewSpecialAuth("root", "Huawei12#$"),
		NewSpecialAuth("root", "Changeme@321"),
		NewSpecialAuth("root", "Changeme123"),
		NewSpecialAuth("sa", "Changeme123"),
		NewSpecialAuth("db2inst1", "db2inst1"),
		NewSpecialAuth("db2fenc1", "db2fenc1"),
		NewSpecialAuth("dasusr1", "dasusr1"),
		NewSpecialAuth("db2admin", "db2admin"),
	}
	return a
}
