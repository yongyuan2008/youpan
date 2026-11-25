import Foundation
import Combine

enum EventKind: String, CaseIterable, Identifiable {
    case retirement, winter, summer, labor, national
    var id: String { rawValue }
    var title: String {
        switch self {
        case .retirement: return "退休"
        case .winter: return "寒假"
        case .summer: return "暑假"
        case .labor: return "五一假期"
        case .national: return "十一假期"
        }
    }
}

struct EventDates: Codable, Equatable {
    var start: String
    var end: String?
    init(start: String, end: String? = nil) { self.start = start; self.end = end }
}

struct CountdownDisplay {
    var value: String
    var unit: String
    var caption: String
    var dateLine: String
}

enum CountdownMath {
    private static var calendar: Calendar {
        var c = Calendar(identifier: .gregorian)
        c.timeZone = .current
        c.locale = Locale(identifier: "en_US_POSIX")
        return c
    }
    static func ymd(_ date: Date) -> String {
        let c = calendar.dateComponents([.year, .month, .day], from: date)
        return String(format: "%04d-%02d-%02d", c.year!, c.month!, c.day!)
    }
    static func date(_ string: String) -> Date? {
        let parts = string.split(separator: "-", omittingEmptySubsequences: false)
        guard parts.count == 3, parts[0].count == 4, parts[1].count == 2, parts[2].count == 2,
              string.utf8.allSatisfy({ (48...57).contains($0) || $0 == 45 }),
              let y = Int(parts[0]), let m = Int(parts[1]), let d = Int(parts[2]), y >= 1 else { return nil }
        var components = DateComponents()
        components.calendar = calendar; components.timeZone = calendar.timeZone
        components.year = y; components.month = m; components.day = d
        guard let result = calendar.date(from: components), ymd(result) == string else { return nil }
        return result
    }
    static func defaultStart(_ kind: EventKind, today: Date) -> Date? {
        let month: Int
        switch kind { case .labor: month = 5; case .national: month = 10; default: return nil }
        let c = calendar
        let now = c.startOfDay(for: today)
        let year = c.component(.year, from: now)
        guard let thisYear = date(String(format: "%04d-%02d-01", year, month)) else { return nil }
        if thisYear >= now { return thisYear }
        return date(String(format: "%04d-%02d-01", year + 1, month))
    }
    static func validate(_ kind: EventKind, dates: EventDates) -> String? {
        guard let start = date(dates.start) else { return "请输入有效的开始日期（YYYY-MM-DD）" }
        if let endString = dates.end {
            guard let end = date(endString) else { return "请输入有效的结束日期（YYYY-MM-DD）" }
            guard end >= start else { return "结束日期不能早于开始日期" }
            if kind == .retirement { return "退休日期无需填写结束日期" }
        }
        return nil
    }
    static func display(_ kind: EventKind, custom: EventDates?, today: Date, customName: String? = nil) -> CountdownDisplay {
        let c = calendar
        let now = c.startOfDay(for: today)
        let start: Date
        let end: Date?
        if let custom = custom {
            guard validate(kind, dates: custom) == nil, let parsed = date(custom.start) else {
                return CountdownDisplay(value: "待设置", unit: "", caption: "日期无效，请重新设置", dateLine: "")
            }
            start = parsed; end = custom.end.flatMap(date)
        } else if let defaultDate = defaultStart(kind, today: today) {
            start = defaultDate; end = nil
        } else {
            if customName != nil {
                return CountdownDisplay(value: "待设置", unit: "", caption: "设置日期", dateLine: "")
            }
            return CountdownDisplay(value: "待设置", unit: "", caption: kind == .retirement ? "设置退休日期" : "设置假期日期", dateLine: "")
        }
        let line = end.map { "\(ymd(start)) 至 \(ymd($0))" } ?? ymd(start)
        let difference = c.dateComponents([.day], from: now, to: start).day ?? 0
        if let customName {
            if difference > 0 {
                return CountdownDisplay(value: String(difference), unit: "天", caption: "距\(customName)", dateLine: line)
            }
            if let end {
                if now <= end {
                    let day = c.dateComponents([.day], from: start, to: now).day! + 1
                    return CountdownDisplay(value: "第 \(day) 天", unit: "", caption: "进行中", dateLine: line)
                }
                let days = c.dateComponents([.day], from: end, to: now).day!
                return CountdownDisplay(value: String(days), unit: "天", caption: "已结束", dateLine: line)
            }
            if difference == 0 { return CountdownDisplay(value: "今天", unit: "", caption: "今天", dateLine: line) }
            return CountdownDisplay(value: String(-difference), unit: "天", caption: "已结束", dateLine: line)
        }
        if difference > 0 {
            return CountdownDisplay(value: String(difference), unit: "天", caption: kind == .retirement ? "距退休" : (custom == nil ? "距节日" : "距放假"), dateLine: line)
        }
        if kind == .retirement {
            if difference == 0 { return CountdownDisplay(value: "今天", unit: "", caption: "退休日", dateLine: line) }
            return CountdownDisplay(value: String(-difference), unit: "天", caption: "已退休", dateLine: line)
        }
        if let end = end {
            if now <= end {
                let day = c.dateComponents([.day], from: start, to: now).day! + 1
                return CountdownDisplay(value: "第 \(day) 天", unit: "", caption: "假期中", dateLine: line)
            }
            let days = c.dateComponents([.day], from: end, to: now).day!
            return CountdownDisplay(value: String(days), unit: "天", caption: "假期已结束", dateLine: line)
        }
        if difference == 0 { return CountdownDisplay(value: "今天", unit: "", caption: custom == nil ? "节日当天" : "假期开始日", dateLine: line) }
        return CountdownDisplay(value: String(-difference), unit: "天", caption: "开始日期已过", dateLine: line)
    }
}

final class CountdownStore: ObservableObject {
    @Published private(set) var selections: [String: EventDates] = [:]
    @Published private(set) var pinned = false
    @Published private(set) var loadWarning: String?
    private let url: URL
    private var loadFailed = false
    private struct DataFile: Codable { var selections: [String: EventDates]; var pinned: Bool }
    enum StoreError: LocalizedError {
        case invalid(String), unreadable
        var errorDescription: String? {
            switch self { case .invalid(let message): return message; case .unreadable: return "原有数据读取失败，为避免覆盖，请先处理存储文件。" }
        }
    }
    convenience init() {
        if let path = ProcessInfo.processInfo.environment["RIZI_COUNTDOWN_TEST_DIRECTORY"], !path.isEmpty {
            self.init(url: URL(fileURLWithPath: path, isDirectory: true).appendingPathComponent("data.json"))
        } else {
            let base = FileManager.default.urls(for: .applicationSupportDirectory, in: .userDomainMask).first!
            self.init(url: base.appendingPathComponent("DaysCountdown", isDirectory: true).appendingPathComponent("data.json"))
        }
    }
    init(url: URL) {
        self.url = url
        do {
            let data = try Data(contentsOf: url)
            let file = try JSONDecoder().decode(DataFile.self, from: data)
            for (key, dates) in file.selections {
                guard let kind = EventKind(rawValue: key), CountdownMath.validate(kind, dates: dates) == nil else {
                    throw StoreError.invalid("已保存的日期无效")
                }
            }
            selections = file.selections; pinned = file.pinned
        } catch let error as NSError where error.domain == NSCocoaErrorDomain && error.code == NSFileReadNoSuchFileError {
            selections = [
                EventKind.retirement.rawValue: EventDates(start: "2046-01-10"),
                EventKind.winter.rawValue: EventDates(start: "2027-01-18", end: "2027-02-18"),
                EventKind.summer.rawValue: EventDates(start: "2027-07-12", end: "2027-08-26")
            ]
        } catch {
            loadFailed = true
            loadWarning = "无法读取已保存的数据：\(error.localizedDescription)"
        }
    }
    func update(_ kind: EventKind, dates: EventDates?) throws {
        if let dates = dates, let message = CountdownMath.validate(kind, dates: dates) { throw StoreError.invalid(message) }
        var next = selections
        next[kind.rawValue] = dates
        try save(selections: next, pinned: pinned)
        selections = next
    }
    func setPinned(_ value: Bool) throws {
        try save(selections: selections, pinned: value)
        pinned = value
    }
    private func save(selections: [String: EventDates], pinned: Bool) throws {
        guard !loadFailed else { throw StoreError.unreadable }
        let data = try JSONEncoder().encode(DataFile(selections: selections, pinned: pinned))
        try FileManager.default.createDirectory(at: url.deletingLastPathComponent(), withIntermediateDirectories: true)
        try data.write(to: url, options: .atomic)
    }
}
