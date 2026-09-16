import Foundation

/// Where the backend lives. A value rather than a global constant so tests can
/// point the client at a stub host and assert that media URLs resolve against
/// whatever base the client was built with, not a hard-coded production host.
struct APIConfig: Sendable, Equatable {
    var baseURL: URL

    // Force-unwrap is safe: the literal is a valid absolute URL, and a typo
    // here should crash in development rather than fail every request quietly.
    static let production = APIConfig(baseURL: URL(string: "https://tarot.sorapure.fun")!)

    /// Builds an endpoint URL. `appending(path:)` keeps any path prefix the base
    /// URL may carry (a reverse-proxy mount), which `URL(string:relativeTo:)`
    /// would silently drop for a path starting with "/".
    func endpointURL(_ path: String) -> URL {
        baseURL.appending(path: path)
    }

    /// Resolves a media reference from the API. The server sends deck art as
    /// server-relative paths (`/media/decks/classic/the_fool.jpg`) so the same
    /// payload works behind any host; an empty string means "no image" and maps
    /// to nil so views can show a placeholder instead of a broken request.
    func mediaURL(_ reference: String?) -> URL? {
        guard let raw = reference?.trimmingCharacters(in: .whitespacesAndNewlines), !raw.isEmpty else {
            return nil
        }
        if let absolute = URL(string: raw), let scheme = absolute.scheme?.lowercased(),
           scheme == "http" || scheme == "https" {
            return absolute
        }
        let path = raw.hasPrefix("/") ? raw : "/" + raw
        if let url = URL(string: path, relativeTo: baseURL)?.absoluteURL {
            return url
        }
        // A file name with a space or non-ASCII character is not a valid URL
        // string as-is; encode rather than drop the image.
        guard let encoded = path.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed) else {
            return nil
        }
        return URL(string: encoded, relativeTo: baseURL)?.absoluteURL
    }
}
