import ImageIO
import SwiftUI
import UIKit

/// Where a card picture comes from.
enum CardArtSource: Hashable, Sendable {
    /// A file in the bundled classic deck, e.g. `the_fool.jpeg`.
    case bundled(String)
    /// A downloaded deck's art, with the classic file to show if it fails.
    case remote(URL, fallbackFile: String)

    var fallbackFile: String {
        switch self {
        case let .bundled(file): return file
        case let .remote(_, fallbackFile): return fallbackFile
        }
    }
}

/// Loads card art, downsampled to the size it is shown at.
///
/// The originals are about 1024×1536. Decoding ten of them at full size for a
/// table of 60-pixel cards would cost tens of megabytes; ImageIO thumbnails
/// decode straight to the target size instead. Downloaded art is also kept on
/// disk by `URLCache`.
final class CardArt: @unchecked Sendable {
    static let shared = CardArt()

    static let backFile = "card_back.jpeg"

    // NSCache is thread-safe, which is what makes the unchecked Sendable sound.
    private let cache = NSCache<NSString, UIImage>()
    private let session: URLSession = {
        let configuration = URLSessionConfiguration.default
        configuration.urlCache = URLCache(memoryCapacity: 16 << 20, diskCapacity: 256 << 20)
        configuration.requestCachePolicy = .returnCacheDataElseLoad
        return URLSession(configuration: configuration)
    }()

    /// Size buckets, so nearby sizes share one cache entry.
    private static func bucket(_ maxPixels: CGFloat) -> Int {
        [256, 512, 1024, 1600].first { CGFloat($0) >= maxPixels } ?? 1600
    }

    func image(file: String, maxPixels: CGFloat) -> UIImage? {
        let bucket = Self.bucket(maxPixels)
        let key = "\(file)@\(bucket)" as NSString
        if let cached = cache.object(forKey: key) {
            return cached
        }
        let name = (file as NSString).deletingPathExtension
        let ext = (file as NSString).pathExtension
        guard let url = Bundle.main.url(forResource: name, withExtension: ext, subdirectory: "cards"),
              let source = CGImageSourceCreateWithURL(url as CFURL, nil),
              let image = Self.thumbnail(source, bucket: bucket) else {
            return nil
        }
        cache.setObject(image, forKey: key)
        return image
    }

    /// A downloaded image if it is already in memory — lets a view, or an
    /// `ImageRenderer` snapshot, show it without waiting.
    func cachedImage(url: URL, maxPixels: CGFloat) -> UIImage? {
        cache.object(forKey: "\(url.absoluteString)@\(Self.bucket(maxPixels))" as NSString)
    }

    func image(url: URL, maxPixels: CGFloat) async -> UIImage? {
        let bucket = Self.bucket(maxPixels)
        let key = "\(url.absoluteString)@\(bucket)" as NSString
        if let cached = cache.object(forKey: key) {
            return cached
        }
        guard let (data, response) = try? await session.data(from: url),
              (response as? HTTPURLResponse)?.statusCode ?? 200 < 400,
              let source = CGImageSourceCreateWithData(data as CFData, nil),
              let image = Self.thumbnail(source, bucket: bucket) else {
            return nil
        }
        cache.setObject(image, forKey: key)
        return image
    }

    private static func thumbnail(_ source: CGImageSource, bucket: Int) -> UIImage? {
        let options: [CFString: Any] = [
            kCGImageSourceCreateThumbnailFromImageAlways: true,
            kCGImageSourceCreateThumbnailWithTransform: true,
            kCGImageSourceShouldCacheImmediately: true,
            kCGImageSourceThumbnailMaxPixelSize: bucket,
        ]
        guard let cgImage = CGImageSourceCreateThumbnailAtIndex(source, 0, options as CFDictionary) else {
            return nil
        }
        return UIImage(cgImage: cgImage)
    }
}

/// A card face or back at a given width, keeping the 2:3 tarot proportions.
struct CardImage: View {
    @Environment(\.displayScale) private var displayScale
    @Environment(\.appColors) private var colors

    let source: CardArtSource
    let width: CGFloat
    /// `.fill` crops to the frame; `.fit` shows the whole picture.
    var contentMode: ContentMode = .fill

    @State private var loaded: UIImage?
    @State private var failed = false

    init(source: CardArtSource, width: CGFloat, contentMode: ContentMode = .fill) {
        self.source = source
        self.width = width
        self.contentMode = contentMode
    }

    /// The bundled classic art.
    init(file: String, width: CGFloat) {
        self.init(source: .bundled(file), width: width)
    }

    var body: some View {
        let height = width * CardMetrics.aspect
        let pixels = height * displayScale
        Group {
            if let image = currentImage(pixels: pixels) {
                Image(uiImage: image)
                    .resizable()
                    .aspectRatio(contentMode: contentMode)
            } else {
                colors.panel
            }
        }
        .frame(width: width, height: height)
        .clipShape(RoundedRectangle(cornerRadius: width * 0.08, style: .continuous))
        .task(id: source) {
            guard case let .remote(url, _) = source else { return }
            failed = false
            loaded = await CardArt.shared.image(url: url, maxPixels: pixels)
            failed = loaded == nil
        }
    }

    private func currentImage(pixels: CGFloat) -> UIImage? {
        switch source {
        case let .bundled(file):
            return CardArt.shared.image(file: file, maxPixels: pixels)
        case let .remote(url, fallbackFile):
            if let image = loaded ?? CardArt.shared.cachedImage(url: url, maxPixels: pixels) {
                return image
            }
            // While loading shows the placeholder; after a failure, the classic art.
            return failed ? CardArt.shared.image(file: fallbackFile, maxPixels: pixels) : nil
        }
    }
}

enum CardMetrics {
    /// Card height over width: 165 / 110, as in the React Native client.
    static let aspect: CGFloat = 1.5
}
