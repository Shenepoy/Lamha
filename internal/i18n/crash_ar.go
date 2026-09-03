package i18n

func init() {
	for english, arabic := range map[string]string{
		"Lamha crash report":         "تقرير عطل لمحة",
		"Lamha stopped unexpectedly": "توقفت لمحة بشكل غير متوقع",
		"Lamha detected that its previous process did not shut down normally. The report below is kept locally until you review it.": "اكتشفت لمحة أن العملية السابقة لم تُغلق بشكل طبيعي. يُحفظ التقرير أدناه محليًا حتى تراجعه.",
		"Report saved at:\n%s": "حُفظ التقرير في:\n%s",
		"Review the report before sharing. It may contain file paths and other details from the app log.": "راجع التقرير قبل مشاركته. قد يحتوي على مسارات ملفات وتفاصيل أخرى من سجل التطبيق.",
		"Open report":                            "فتح التقرير",
		"Copy report":                            "نسخ التقرير",
		"Report on GitHub":                       "الإبلاغ على GitHub",
		"Dismiss":                                "تجاهل",
		"Crash report copied to the clipboard.":  "نُسخ تقرير العطل إلى الحافظة.",
		"Could not open crash report: %v":        "تعذّر فتح تقرير العطل: %v",
		"Could not copy crash report: %v":        "تعذّر نسخ تقرير العطل: %v",
		"Could not create a GitHub issue link.":  "تعذّر إنشاء رابط بلاغ GitHub.",
		"Could not open GitHub issue form: %v":   "تعذّر فتح نموذج بلاغ GitHub: %v",
		"Open Lamha to review the crash report.": "افتح لمحة لمراجعة تقرير العطل.",
	} {
		arabicMessages[english] = arabic
	}
}
