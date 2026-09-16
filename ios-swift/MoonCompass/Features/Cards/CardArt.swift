import ImageIO
import SwiftUI
import UIKit

/// Loads the bundled card art, downsampled to the size it is shown at.
///
/// The originals are about 1024×1536. Decoding ten of them at full size for a
/// table of 60-pixel cards would cost tens of megabytes; ImageIO thumbnails
/// decode straight to the target size instead.
final class CardArt: @unchecked Sendable {
    static let shared = CardArt()

    static let backFile = "card_back.jpeg"

    // NSCache is thread-safe, which is what makes the unchecked Sendable sound.
    private let cache = NSCache<NSString, UIImage>()

    /// `maxPixels` is rounded up to a few size buckets so nearby sizes share
    /// one cache entry.
    func image(file: String, maxPixels: CGFloat) -> UIImage? {
        let bucket = [256, 512, 1024, 1600].first { CGFloat($0) >= maxPixels } ?? 1600
        let key = "\(file)@\(bucket)" as NSString
        if let cached = cache.object(forKey: key) {
            return cached
        }
        let name = (file as NSString).deletingPathExtension
        let ext = (file as NSString).pathExtension
        guard let url = Bundle.main.url(forResource: name, withExtension: ext, subdirectory: "cards"),
              let source = CGImageSourceCreateWithURL(url as CFURL, nil) else {
            return nil
        }
        let options: [CFString: Any] = [
            kCGImageSourceCreateThumbnailFromImageAlways: true,
            kCGImageSourceCreateThumbnailWithTransform: true,
            kCGImageSourceShouldCacheImmediately: true,
            kCGImageSourceThumbnailMaxPixelSize: bucket,
        ]
        guard let cgImage = CGImageSourceCreateThumbnailAtIndex(source, 0, options as CFDictionary) else {
            return nil
        }
        let image = UIImage(cgImage: cgImage)
        cache.setObject(image, forKey: key)
        return image
    }
}

/// A card face or back at a given width, keeping the 2:3 tarot proportions.
struct CardImage: View {
    @Environment(\.displayScale) private var displayScale
    @Environment(\.appColors) private var colors

    let file: String
    let width: CGFloat

    var body: some View {
        let height = width * CardMetrics.aspect
        Group {
            if let image = CardArt.shared.image(file: file, maxPixels: height * displayScale) {
                Image(uiImage: image)
                    .resizable()
                    .scaledToFill()
            } else {
                colors.panel
            }
        }
        .frame(width: width, height: height)
        .clipShape(RoundedRectangle(cornerRadius: width * 0.08, style: .continuous))
    }
}

enum CardMetrics {
    /// Card height over width: 165 / 110, as in the React Native client.
    static let aspect: CGFloat = 1.5
}
