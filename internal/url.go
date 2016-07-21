package internal

import "net/url"

// SafeURL removes sensitive information from a URL.
func SafeURL(u *url.URL) string {
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

// SafeURLFromString removes sensitive information from a URL.
func SafeURLFromString(rawurl string) string {
	u, err := url.Parse(rawurl)
	if nil != err {
		return ""
	}
	return SafeURL(u)
}

// HostFromExternalURL returns the URL's host.
func HostFromExternalURL(rawurl string) string {
	u, err := url.Parse(rawurl)
	if nil != err {
		return ""
	}
	if "" != u.Opaque {
		return ""
	}
	return u.Host
}
