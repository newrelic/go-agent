package internal

import "net/url"

func safeURL(u *url.URL) string {
	if "" != u.Opaque {
		// If the URL is opaque, we cannot be sure if it contains
		// sensitive information.
		return ""
	}

	// Omit user, query, and fragment information for security.
	ur := url.URL{
		Scheme: u.Scheme,
		Host:   u.Host,
		Path:   u.Path,
	}
	return ur.String()
}

func safeURLFromString(rawurl string) string {
	u, err := url.Parse(rawurl)
	if nil != err {
		return ""
	}
	return safeURL(u)
}
