import SwiftUI

/// Pinch to zoom and drag to pan the reading table — a port of the React
/// Native `ZoomableView`. Zoom is anchored where the fingers are, snaps back
/// when barely zoomed, and panning is kept within the zoomed content.
struct ZoomableCanvas<Content: View>: View {
    @Environment(\.appColors) private var colors

    var isEnabled: Bool
    /// Changing it (e.g. after a redraw) resets the zoom.
    var resetKey: String
    var hint: String?
    var resetLabel: String
    @ViewBuilder var content: Content

    static var maxScale: CGFloat { 3.5 }
    /// Below this the zoom counts as "not zoomed" and snaps back on release.
    static var snapThreshold: CGFloat { 1.04 }

    @State private var scale: CGFloat = 1
    @State private var offset: CGSize = .zero
    @State private var baseScale: CGFloat = 1
    @State private var baseOffset: CGSize = .zero

    private var isZoomed: Bool { scale > Self.snapThreshold }

    var body: some View {
        GeometryReader { geometry in
            let size = geometry.size
            content
                .frame(width: size.width, height: size.height)
                .scaleEffect(scale)
                .offset(offset)
                .clipped()
                .contentShape(Rectangle())
                .gesture(pinch(size: size), including: isEnabled ? .all : .subviews)
                .simultaneousGesture(pan(size: size), including: isEnabled && isZoomed ? .all : .subviews)
                .overlay(alignment: .topTrailing) {
                    if isEnabled, isZoomed {
                        Button(resetLabel) { reset(animated: true) }
                            .font(.caption.weight(.semibold))
                            .foregroundStyle(colors.text)
                            .padding(.horizontal, 10)
                            .padding(.vertical, 6)
                            .background(colors.bg.opacity(0.75), in: Capsule())
                            .padding(8)
                    }
                }
                .overlay(alignment: .bottom) {
                    if isEnabled, !isZoomed, let hint {
                        Text(hint)
                            .font(.caption)
                            .foregroundStyle(colors.muted)
                            .padding(.horizontal, 10)
                            .padding(.vertical, 4)
                            .background(colors.bg.opacity(0.6), in: Capsule())
                            .padding(.bottom, 6)
                            .allowsHitTesting(false)
                    }
                }
        }
        .onChange(of: resetKey) { reset(animated: false) }
    }

    private func pinch(size: CGSize) -> some Gesture {
        MagnifyGesture()
            .onChanged { value in
                let newScale = min(Self.maxScale, max(1, baseScale * value.magnification))
                // Keep the point between the fingers under the fingers.
                let focal = CGPoint(x: value.startAnchor.x * size.width, y: value.startAnchor.y * size.height)
                let ratio = newScale / baseScale
                offset = CGSize(
                    width: baseOffset.width + (focal.x - size.width / 2 - baseOffset.width) * (1 - ratio),
                    height: baseOffset.height + (focal.y - size.height / 2 - baseOffset.height) * (1 - ratio)
                )
                scale = newScale
            }
            .onEnded { _ in settle(size: size) }
    }

    private func pan(size: CGSize) -> some Gesture {
        DragGesture(minimumDistance: 12)
            .onChanged { value in
                guard isZoomed else { return }
                offset = CGSize(
                    width: baseOffset.width + value.translation.width,
                    height: baseOffset.height + value.translation.height
                )
            }
            .onEnded { _ in settle(size: size) }
    }

    private func settle(size: CGSize) {
        guard isZoomed else {
            reset(animated: true)
            return
        }
        // Content scaled around its centre can move at most half its growth
        // in each direction before an edge comes into view.
        let maxX = (scale - 1) * size.width / 2
        let maxY = (scale - 1) * size.height / 2
        withAnimation(.easeOut(duration: 0.2)) {
            offset = CGSize(
                width: min(maxX, max(-maxX, offset.width)),
                height: min(maxY, max(-maxY, offset.height))
            )
        }
        baseScale = scale
        baseOffset = offset
    }

    private func reset(animated: Bool) {
        let apply = {
            scale = 1
            offset = .zero
        }
        if animated {
            withAnimation(.easeOut(duration: 0.2), apply)
        } else {
            apply()
        }
        baseScale = 1
        baseOffset = .zero
    }
}
