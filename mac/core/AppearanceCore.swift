import Foundation
import Combine
import CoreGraphics

struct AppearanceConfiguration: Codable, Equatable {
    var width: Double
    var height: Double
    var opacity: Double
    var names: [String: String]
    var pinnedToDesktop: Bool

    private enum CodingKeys: String, CodingKey {
        case width, height, opacity, names, pinnedToDesktop
    }

    init(width: Double, height: Double, opacity: Double, names: [String: String], pinnedToDesktop: Bool) {
        self.width = width
        self.height = height
        self.opacity = opacity
        self.names = names
        self.pinnedToDesktop = pinnedToDesktop
    }

    init(from decoder: Decoder) throws {
        let values = try decoder.container(keyedBy: CodingKeys.self)
        width = try values.decode(Double.self, forKey: .width)
        height = try values.decode(Double.self, forKey: .height)
        opacity = try values.decode(Double.self, forKey: .opacity)
        names = try values.decode([String: String].self, forKey: .names)
        // v2.2 and older appearance files do not contain this key.
        pinnedToDesktop = try values.decodeIfPresent(Bool.self, forKey: .pinnedToDesktop) ?? false
    }

    static let defaultWidth = 760.0
    static let defaultHeight = 640.0
    static let defaultOpacity = 1.0
    static let minimumWidth = 520.0
    static let minimumHeight = 440.0
    static let minimumLayoutWidth = 700.0
    static let minimumLayoutHeight = 590.0
    static let maximumDimension = 4096.0

    static var `default`: AppearanceConfiguration {
        AppearanceConfiguration(width: defaultWidth, height: defaultHeight, opacity: defaultOpacity, names: [:], pinnedToDesktop: false)
    }
}

enum DesktopPanelGeometry {
    static let defaultSize = CGSize(width: 316, height: 318)

    static func cappedFrame(_ frame: CGRect, visibleFrames: [CGRect], fallback: CGRect) -> CGRect {
        let candidates = visibleFrames.filter { $0.width > 0 && $0.height > 0 }
        let screens = candidates.isEmpty ? [fallback] : candidates
        let center = CGPoint(x: frame.midX, y: frame.midY)
        let target = screens.max { lhs, rhs in
            let lhsArea = intersectionArea(frame, lhs)
            let rhsArea = intersectionArea(frame, rhs)
            if lhsArea != rhsArea { return lhsArea < rhsArea }
            return squaredDistance(center, CGPoint(x: lhs.midX, y: lhs.midY)) > squaredDistance(center, CGPoint(x: rhs.midX, y: rhs.midY))
        } ?? fallback
        var result = frame
        result.size.width = min(max(1, frame.width), target.width)
        result.size.height = min(max(1, frame.height), target.height)
        result.origin.x = min(max(frame.minX, target.minX), target.maxX - result.width)
        result.origin.y = min(max(frame.minY, target.minY), target.maxY - result.height)
        return result
    }

    private static func intersectionArea(_ lhs: CGRect, _ rhs: CGRect) -> CGFloat {
        let intersection = lhs.intersection(rhs)
        return intersection.isNull ? 0 : intersection.width * intersection.height
    }

    private static func squaredDistance(_ lhs: CGPoint, _ rhs: CGPoint) -> CGFloat {
        let x = lhs.x - rhs.x, y = lhs.y - rhs.y
        return x * x + y * y
    }
}

enum WindowGeometry {
    static func viewportSize(requested: CGSize, available: CGSize) -> CGSize {
        let availableWidth = available.width.isFinite && available.width > 0 ? available.width : AppearanceConfiguration.minimumWidth
        let availableHeight = available.height.isFinite && available.height > 0 ? available.height : AppearanceConfiguration.minimumHeight
        let requestedWidth = requested.width.isFinite && requested.width > 0 ? requested.width : AppearanceConfiguration.defaultWidth
        let requestedHeight = requested.height.isFinite && requested.height > 0 ? requested.height : AppearanceConfiguration.defaultHeight
        return CGSize(width: min(requestedWidth, availableWidth), height: min(requestedHeight, availableHeight))
    }

    static func contentMinimum(available: CGSize) -> CGSize {
        let viewport = viewportSize(
            requested: CGSize(width: AppearanceConfiguration.minimumWidth, height: AppearanceConfiguration.minimumHeight),
            available: available
        )
        return CGSize(width: max(1, viewport.width), height: max(1, viewport.height))
    }

    static func persistedSize(actual: CGSize) -> CGSize {
        CGSize(
            width: min(AppearanceConfiguration.maximumDimension, max(AppearanceConfiguration.minimumWidth, actual.width)),
            height: min(AppearanceConfiguration.maximumDimension, max(AppearanceConfiguration.minimumHeight, actual.height))
        )
    }

    static func fitScale(viewport: CGSize) -> CGFloat {
        guard viewport.width.isFinite, viewport.height.isFinite, viewport.width > 0, viewport.height > 0 else { return 1 }
        return min(
            1,
            viewport.width / AppearanceConfiguration.minimumLayoutWidth,
            viewport.height / AppearanceConfiguration.minimumLayoutHeight
        )
    }

    static func fittedLayoutSize(viewport: CGSize) -> CGSize {
        let scale = fitScale(viewport: viewport)
        guard scale > 0 else {
            return CGSize(width: AppearanceConfiguration.minimumLayoutWidth, height: AppearanceConfiguration.minimumLayoutHeight)
        }
        return CGSize(
            width: max(AppearanceConfiguration.minimumLayoutWidth, viewport.width / scale),
            height: max(AppearanceConfiguration.minimumLayoutHeight, viewport.height / scale)
        )
    }
}

final class AppearanceStore: ObservableObject {
    @Published private(set) var configuration = AppearanceConfiguration.default
    @Published private(set) var loadWarning: String?

    private let url: URL
    private var loadFailed = false

    enum StoreError: LocalizedError {
        case invalid(String)
        case unreadable

        var errorDescription: String? {
            switch self {
            case .invalid(let message): return message
            case .unreadable: return "原有外观配置读取失败，为避免覆盖，请先处理 appearance.json。"
            }
        }
    }

    convenience init() {
        if let path = ProcessInfo.processInfo.environment["RIZI_COUNTDOWN_TEST_DIRECTORY"], !path.isEmpty {
            self.init(url: URL(fileURLWithPath: path, isDirectory: true).appendingPathComponent("appearance.json"))
        } else {
            let base = FileManager.default.urls(for: .applicationSupportDirectory, in: .userDomainMask).first!
            self.init(url: base.appendingPathComponent("DaysCountdown", isDirectory: true).appendingPathComponent("appearance.json"))
        }
    }

    init(url: URL) {
        self.url = url
        do {
            let data = try Data(contentsOf: url)
            var decoded = try JSONDecoder().decode(AppearanceConfiguration.self, from: data)
            decoded.names = try Self.normalizedNames(decoded.names)
            try Self.validateDimensions(width: decoded.width, height: decoded.height)
            try Self.validateOpacity(decoded.opacity)
            configuration = decoded
        } catch let error as NSError where error.domain == NSCocoaErrorDomain && error.code == NSFileReadNoSuchFileError {
            configuration = .default
        } catch {
            loadFailed = true
            loadWarning = "无法读取已保存的外观配置：\(error.localizedDescription)"
        }
    }

    var width: Double { configuration.width }
    var height: Double { configuration.height }
    var opacity: Double { configuration.opacity }
    var pinnedToDesktop: Bool { configuration.pinnedToDesktop }

    func customName(for kind: EventKind) -> String? { configuration.names[kind.rawValue] }
    func name(for kind: EventKind) -> String { customName(for: kind) ?? kind.title }

    func updateSize(width: Double, height: Double) throws {
        try Self.validateDimensions(width: width, height: height)
        var next = configuration
        next.width = width
        next.height = height
        try persist(next)
    }

    func updateOpacity(_ opacity: Double) throws {
        try Self.validateOpacity(opacity)
        var next = configuration
        next.opacity = opacity
        try persist(next)
    }

    func setPinnedToDesktop(_ value: Bool) throws {
        var next = configuration
        next.pinnedToDesktop = value
        try persist(next)
    }

    func resetAppearance() throws {
        var next = configuration
        next.width = AppearanceConfiguration.defaultWidth
        next.height = AppearanceConfiguration.defaultHeight
        next.opacity = AppearanceConfiguration.defaultOpacity
        next.pinnedToDesktop = false
        try persist(next)
    }

    func updateNames(_ names: [String: String]) throws {
        var next = configuration
        next.names = try Self.normalizedNames(names)
        try persist(next)
    }

    func resetNames() throws {
        var next = configuration
        next.names = [:]
        try persist(next)
    }

    private func persist(_ next: AppearanceConfiguration) throws {
        guard !loadFailed else { throw StoreError.unreadable }
        try Self.validateDimensions(width: next.width, height: next.height)
        try Self.validateOpacity(next.opacity)
        _ = try Self.normalizedNames(next.names)
        let data = try JSONEncoder().encode(next)
        try FileManager.default.createDirectory(at: url.deletingLastPathComponent(), withIntermediateDirectories: true)
        try data.write(to: url, options: .atomic)
        configuration = next
    }

    static func normalizedName(_ value: String) throws -> String {
        let trimmed = value.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { throw StoreError.invalid("倒计时名称不能为空") }
        guard trimmed.count <= 24 else { throw StoreError.invalid("倒计时名称不能超过 24 个字符") }
        return trimmed
    }

    private static func normalizedNames(_ names: [String: String]) throws -> [String: String] {
        var result: [String: String] = [:]
        for (key, value) in names {
            guard let kind = EventKind(rawValue: key) else { throw StoreError.invalid("外观配置包含未知的倒计时名称") }
            let normalized = try normalizedName(value)
            if normalized != kind.title { result[key] = normalized }
        }
        return result
    }

    private static func validateDimensions(width: Double, height: Double) throws {
        guard width.isFinite, height.isFinite,
              width >= AppearanceConfiguration.minimumWidth,
              height >= AppearanceConfiguration.minimumHeight,
              width <= AppearanceConfiguration.maximumDimension,
              height <= AppearanceConfiguration.maximumDimension else {
            throw StoreError.invalid("窗口尺寸超出允许范围")
        }
    }

    private static func validateOpacity(_ opacity: Double) throws {
        guard opacity.isFinite, (0.5...1.0).contains(opacity) else {
            throw StoreError.invalid("窗口透明度必须在 50% 到 100% 之间")
        }
    }
}
