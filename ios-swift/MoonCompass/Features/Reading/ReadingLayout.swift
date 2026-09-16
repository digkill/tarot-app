import CoreGraphics

/// Card sizing and placement on the reading table — a port of
/// `computeCardWidth` from the React Native `ReadingScreen`.
enum ReadingLayout {
    static let baseCardWidth: CGFloat = 110
    static let labelHeight: CGFloat = 24
    static let padding: CGFloat = 8
    static let minimumCardWidth: CGFloat = 36

    /// Table height by spread size: 440 for 10+ cards, 400 for 7+, else 320.
    static func canvasHeight(cardCount: Int) -> CGFloat {
        cardCount >= 10 ? 440 : cardCount >= 7 ? 400 : 320
    }

    /// The largest card width at which neighbours don't collide.
    ///
    /// Looks at the tightest horizontal gap between cards on roughly the same
    /// row and the tightest vertical gap between cards in roughly the same
    /// column, plus a rows-and-columns estimate from the spread's extent, and
    /// takes the smallest positive candidate, never above the base width.
    static func cardWidth(positions: [SpreadPosition], canvas: CGSize) -> CGFloat {
        guard positions.count > 1 else {
            return min(baseCardWidth, canvas.width - padding * 2)
        }
        let aspect = CardMetrics.aspect

        var minDeltaX = CGFloat.infinity
        var minDeltaY = CGFloat.infinity
        for i in positions.indices {
            for j in positions.indices where j > i {
                let dx = abs(positions[i].x - positions[j].x) * canvas.width
                let dy = abs(positions[i].y - positions[j].y) * canvas.height
                if dy < 20, dx > 1 { minDeltaX = min(minDeltaX, dx) }
                if dx < 20, dy > 1 { minDeltaY = min(minDeltaY, dy) }
            }
        }

        var candidates: [CGFloat] = []
        if minDeltaX.isFinite {
            candidates.append(minDeltaX * 0.85)
        }
        if minDeltaY.isFinite {
            candidates.append((minDeltaY * 0.85 - labelHeight) / aspect)
        }

        let xs = positions.map(\.x)
        let ys = positions.map(\.y)
        let usableWidth = canvas.width - padding * 2
        let usableHeight = canvas.height - padding * 2
        let xSpan = max((xs.max() ?? 0) - (xs.min() ?? 0), 0.01)
        let ySpan = max((ys.max() ?? 0) - (ys.min() ?? 0), 0.01)
        let step = 1 / Double(positions.count + 1)
        let columns = xSpan / step + 1
        let rows = ySpan / step + 1
        candidates.append(usableWidth / max(columns, 1) * 0.85)
        candidates.append((usableHeight / max(rows, 1) - labelHeight) / aspect * 0.85)

        let computed = candidates.reduce(baseCardWidth) { current, value in
            value > 0 && value < current ? value : current
        }
        return max(minimumCardWidth, computed.rounded())
    }

    /// Top-left corner of a card, kept inside the table with room for its label.
    static func origin(for position: SpreadPosition, cardWidth: CGFloat, canvas: CGSize) -> CGPoint {
        let cardHeight = (cardWidth * CardMetrics.aspect).rounded()
        let rawX = canvas.width * position.x - cardWidth / 2
        let rawY = canvas.height * position.y - cardHeight / 2
        let x = max(padding, min(rawX, canvas.width - cardWidth - padding))
        let y = max(padding, min(rawY, canvas.height - cardHeight - labelHeight - padding))
        return CGPoint(x: x, y: y)
    }
}
