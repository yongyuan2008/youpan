import Foundation
import Combine

enum TaskRecurrence: String, Codable, CaseIterable, Identifiable {
    case none, daily, weekdays, weekly
    var id: String { rawValue }
    var title: String {
        switch self {
        case .none: return "不重复"
        case .daily: return "每天"
        case .weekdays: return "工作日"
        case .weekly: return "每周"
        }
    }
}

struct PlannerTask: Codable, Equatable, Identifiable {
    var id: String
    var title: String
    var details: String
    var startDate: String?
    var reminderTime: String?
    var recurrence: TaskRecurrence
    var isPriority: Bool
    var completedOccurrences: Set<String>
    var sourceNoteID: String?
    var createdAt: String

    init(id: String = UUID().uuidString, title: String, details: String = "", startDate: String? = nil,
         reminderTime: String? = nil, recurrence: TaskRecurrence = .none, isPriority: Bool = false,
         completedOccurrences: Set<String> = [], sourceNoteID: String? = nil,
         createdAt: String = PlannerClock.timestamp()) {
        self.id = id; self.title = title; self.details = details; self.startDate = startDate
        self.reminderTime = reminderTime; self.recurrence = recurrence; self.isPriority = isPriority
        self.completedOccurrences = completedOccurrences; self.sourceNoteID = sourceNoteID; self.createdAt = createdAt
    }
}

struct PlannerNote: Codable, Equatable, Identifiable {
    var id: String
    var title: String
    var body: String
    var createdAt: String
    var updatedAt: String

    init(id: String = UUID().uuidString, title: String, body: String,
         createdAt: String = PlannerClock.timestamp(), updatedAt: String = PlannerClock.timestamp()) {
        self.id = id; self.title = title; self.body = body; self.createdAt = createdAt; self.updatedAt = updatedAt
    }
}

enum PlannerClock {
    static func timestamp(_ date: Date = Date()) -> String {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        return formatter.string(from: date)
    }
}

enum PlannerValidation {
    static func validTime(_ value: String) -> Bool {
        let parts = value.split(separator: ":", omittingEmptySubsequences: false)
        guard parts.count == 2, parts[0].count == 2, parts[1].count == 2,
              value.utf8.enumerated().allSatisfy({ index, byte in index == 2 ? byte == 58 : (48...57).contains(byte) }),
              let hour = Int(parts[0]), let minute = Int(parts[1]) else { return false }
        return (0...23).contains(hour) && (0...59).contains(minute)
    }

    static func validate(_ task: PlannerTask) -> String? {
        if task.title.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty { return "待办标题不能为空" }
        if let date = task.startDate, CountdownMath.date(date) == nil { return "待办日期无效" }
        if let time = task.reminderTime, !validTime(time) { return "提醒时间无效" }
        if task.reminderTime != nil && task.startDate == nil { return "提醒需要先安排日期" }
        if task.recurrence != .none && task.startDate == nil { return "重复待办需要开始日期" }
        return nil
    }
}

enum TaskEditorDefaults {
    /// A calendar-selected date seeds only a brand-new task. Editing an existing unscheduled task
    /// must preserve its unscheduled state unless the user explicitly enables a date.
    static func initialDate(existing: PlannerTask?, suggestedDate: String?) -> String? {
        if let existing { return existing.startDate }
        return suggestedDate
    }
}

final class UnsavedNoteGuard {
    static let shared = UnsavedNoteGuard()
    private(set) var isDirty = false
    private var saver: (() -> Bool)?

    func register(saver: @escaping () -> Bool) { self.saver = saver; isDirty = false }
    func setDirty(_ value: Bool) { isDirty = value }
    func unregister() { saver = nil; isDirty = false }
    func attemptSave() -> Bool {
        guard isDirty else { return true }
        guard let saver, saver() else { return false }
        isDirty = false
        return true
    }
    func discard() { isDirty = false; saver = nil }
}

enum TaskSchedule {
    private static func calendar(_ timeZone: TimeZone = .current) -> Calendar {
        var calendar = Calendar(identifier: .gregorian)
        calendar.timeZone = timeZone
        calendar.locale = Locale(identifier: "en_US_POSIX")
        return calendar
    }

    static func occurs(_ task: PlannerTask, on day: String, timeZone: TimeZone = .current) -> Bool {
        guard let startString = task.startDate,
              let start = date(startString, timeZone: timeZone),
              let target = date(day, timeZone: timeZone), target >= start else { return false }
        let cal = calendar(timeZone)
        switch task.recurrence {
        case .none: return day == startString
        case .daily: return true
        case .weekdays:
            let weekday = cal.component(.weekday, from: target)
            return weekday != 1 && weekday != 7
        case .weekly:
            return cal.component(.weekday, from: target) == cal.component(.weekday, from: start)
        }
    }

    static func occurrenceKey(_ task: PlannerTask, on day: String?) -> String {
        if task.startDate == nil { return "unscheduled" }
        return day ?? task.startDate ?? "unscheduled"
    }

    static func isCompleted(_ task: PlannerTask, on day: String?) -> Bool {
        task.completedOccurrences.contains(occurrenceKey(task, on: day))
    }

    static func date(_ string: String, timeZone: TimeZone) -> Date? {
        let parts = string.split(separator: "-", omittingEmptySubsequences: false)
        guard parts.count == 3, parts[0].count == 4, parts[1].count == 2, parts[2].count == 2,
              let year = Int(parts[0]), let month = Int(parts[1]), let day = Int(parts[2]) else { return nil }
        var components = DateComponents()
        components.calendar = calendar(timeZone); components.timeZone = timeZone
        components.year = year; components.month = month; components.day = day
        guard let value = calendar(timeZone).date(from: components) else { return nil }
        return ymd(value, timeZone: timeZone) == string ? value : nil
    }

    static func ymd(_ date: Date, timeZone: TimeZone = .current) -> String {
        let parts = calendar(timeZone).dateComponents([.year, .month, .day], from: date)
        return String(format: "%04d-%02d-%02d", parts.year!, parts.month!, parts.day!)
    }

    static func todaySections(tasks: [PlannerTask], day: String, timeZone: TimeZone = .current)
        -> (overdue: [PlannerTask], current: [PlannerTask], unscheduled: [PlannerTask], completed: [PlannerTask]) {
        let sorter: (PlannerTask, PlannerTask) -> Bool = {
            if $0.isPriority != $1.isPriority { return $0.isPriority && !$1.isPriority }
            return $0.createdAt < $1.createdAt
        }
        var overdue: [PlannerTask] = [], current: [PlannerTask] = [], unscheduled: [PlannerTask] = [], completed: [PlannerTask] = []
        for task in tasks {
            if task.startDate == nil {
                if isCompleted(task, on: nil) { completed.append(task) } else { unscheduled.append(task) }
            } else if occurs(task, on: day, timeZone: timeZone) {
                if isCompleted(task, on: day) { completed.append(task) } else { current.append(task) }
            } else if task.recurrence == .none, let start = task.startDate, start < day, !isCompleted(task, on: start) {
                overdue.append(task)
            }
        }
        return (overdue.sorted(by: sorter), current.sorted(by: sorter), unscheduled.sorted(by: sorter), completed.sorted(by: sorter))
    }
}

struct CalendarGrid {
    let monthStart: Date
    let days: [Date]

    init(containing date: Date, timeZone: TimeZone = .current) {
        var cal = Calendar(identifier: .gregorian); cal.timeZone = timeZone; cal.locale = Locale(identifier: "zh_CN")
        let components = cal.dateComponents([.year, .month], from: date)
        monthStart = cal.date(from: components)!
        let weekday = cal.component(.weekday, from: monthStart)
        let mondayOffset = (weekday + 5) % 7
        let gridStart = cal.date(byAdding: .day, value: -mondayOffset, to: monthStart)!
        days = (0..<42).map { cal.date(byAdding: .day, value: $0, to: gridStart)! }
    }
}

struct AgendaOccurrence: Equatable, Identifiable {
    let task: PlannerTask
    let day: String
    let completed: Bool
    var id: String { "\(task.id)|\(day)" }
}

struct AgendaDayGroup: Equatable, Identifiable {
    let day: String
    let occurrences: [AgendaOccurrence]
    var id: String { day }
}

enum AgendaSelector {
    static func matches(_ task: PlannerTask, query: String, notes: [PlannerNote]) -> Bool {
        let needle = query.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !needle.isEmpty else { return true }
        if task.title.localizedCaseInsensitiveContains(needle) || task.details.localizedCaseInsensitiveContains(needle) { return true }
        guard let sourceID = task.sourceNoteID,
              let note = notes.first(where: { $0.id == sourceID }) else { return false }
        return note.title.localizedCaseInsensitiveContains(needle)
    }

    /// Selects real occurrences in [startDay, startDay + dayCount), never cloning or persisting tasks.
    /// Unscheduled tasks are intentionally excluded because they do not have calendar occurrences.
    static func upcoming(tasks: [PlannerTask], notes: [PlannerNote], startDay: String, dayCount: Int,
                         query: String = "", timeZone: TimeZone = .current) -> [AgendaDayGroup] {
        guard dayCount > 0, let start = TaskSchedule.date(startDay, timeZone: timeZone) else { return [] }
        var calendar = Calendar(identifier: .gregorian); calendar.timeZone = timeZone
        let visibleTasks = tasks.filter { $0.startDate != nil && matches($0, query: query, notes: notes) }
        return (0..<dayCount).compactMap { offset in
            guard let date = calendar.date(byAdding: .day, value: offset, to: start) else { return nil }
            let day = TaskSchedule.ymd(date, timeZone: timeZone)
            let occurrences = visibleTasks.filter { TaskSchedule.occurs($0, on: day, timeZone: timeZone) }
                .sorted {
                    if $0.isPriority != $1.isPriority { return $0.isPriority && !$1.isPriority }
                    return $0.createdAt < $1.createdAt
                }
                .map { AgendaOccurrence(task: $0, day: day, completed: TaskSchedule.isCompleted($0, on: day)) }
            return occurrences.isEmpty ? nil : AgendaDayGroup(day: day, occurrences: occurrences)
        }
    }
}

struct ReminderEvent: Equatable {
    let taskID: String
    let occurrence: String
    let reminderTime: String
    let title: String
    var key: String { "\(taskID)|\(occurrence)|\(reminderTime)" }
}

enum ReminderLogic {
    /// Catches up reminders from the current local day only. Launching or waking later the same day
    /// still reminds once, while earlier days (including repeating occurrences) never flood forward.
    static func due(tasks: [PlannerTask], delivered: Set<String>, now: Date, timeZone: TimeZone = .current) -> [ReminderEvent] {
        var cal = Calendar(identifier: .gregorian); cal.timeZone = timeZone
        let day = TaskSchedule.ymd(now, timeZone: timeZone)
        return tasks.compactMap { task in
            guard let time = task.reminderTime, PlannerValidation.validTime(time),
                  TaskSchedule.occurs(task, on: day, timeZone: timeZone),
                  !TaskSchedule.isCompleted(task, on: day) else { return nil }
            let pieces = time.split(separator: ":")
            let bits: [Int] = pieces.compactMap { Int(String($0)) }
            guard bits.count == 2, let base = TaskSchedule.date(day, timeZone: timeZone),
                  let due = cal.date(bySettingHour: bits[0], minute: bits[1], second: 0, of: base),
                  due <= now else { return nil }
            let event = ReminderEvent(taskID: task.id, occurrence: day, reminderTime: time, title: task.title)
            return delivered.contains(event.key) ? nil : event
        }.sorted { $0.reminderTime == $1.reminderTime ? $0.title < $1.title : $0.reminderTime < $1.reminderTime }
    }
}

final class PlannerStore: ObservableObject {
    static let shared = PlannerStore()

    @Published private(set) var tasks: [PlannerTask] = []
    @Published private(set) var notes: [PlannerNote] = []
    @Published private(set) var deliveredReminderKeys: Set<String> = []
    @Published private(set) var loadWarning: String?
    private let url: URL
    private var loadFailed = false

    private struct PlannerFile: Codable {
        var schemaVersion: Int
        var tasks: [PlannerTask]
        var notes: [PlannerNote]
        var deliveredReminderKeys: Set<String>
    }

    enum StoreError: LocalizedError {
        case invalid(String), unreadable, missing
        var errorDescription: String? {
            switch self {
            case .invalid(let message): return message
            case .unreadable: return "规划数据读取失败，为避免覆盖，请先处理 planner.json。"
            case .missing: return "记录不存在"
            }
        }
    }

    convenience init() {
        if let path = ProcessInfo.processInfo.environment["RIZI_COUNTDOWN_TEST_DIRECTORY"], !path.isEmpty {
            self.init(url: URL(fileURLWithPath: path, isDirectory: true).appendingPathComponent("planner.json"))
        } else {
            let base = FileManager.default.urls(for: .applicationSupportDirectory, in: .userDomainMask).first!
            self.init(url: base.appendingPathComponent("DaysCountdown", isDirectory: true).appendingPathComponent("planner.json"))
        }
    }

    init(url: URL) {
        self.url = url
        do {
            let data = try Data(contentsOf: url)
            let file = try JSONDecoder().decode(PlannerFile.self, from: data)
            guard file.schemaVersion == 1 else { throw StoreError.invalid("不支持的规划数据版本") }
            guard Set(file.tasks.map(\.id)).count == file.tasks.count,
                  Set(file.notes.map(\.id)).count == file.notes.count else { throw StoreError.invalid("规划数据含重复标识") }
            for task in file.tasks {
                if let error = PlannerValidation.validate(task) { throw StoreError.invalid(error) }
                if let source = task.sourceNoteID, !file.notes.contains(where: { $0.id == source }) {
                    throw StoreError.invalid("待办引用了不存在的笔记")
                }
            }
            tasks = file.tasks; notes = file.notes; deliveredReminderKeys = file.deliveredReminderKeys
        } catch let error as NSError where error.domain == NSCocoaErrorDomain && error.code == NSFileReadNoSuchFileError {
            // First launch intentionally does not create a file until the user changes something.
        } catch {
            loadFailed = true
            loadWarning = "无法读取规划数据：\(error.localizedDescription)"
        }
    }

    func addTask(_ task: PlannerTask) throws {
        if let error = PlannerValidation.validate(task) { throw StoreError.invalid(error) }
        var next = tasks; next.append(task); try save(tasks: next, notes: notes, reminders: deliveredReminderKeys); tasks = next
    }

    func updateTask(_ task: PlannerTask) throws {
        if let error = PlannerValidation.validate(task) { throw StoreError.invalid(error) }
        guard let index = tasks.firstIndex(where: { $0.id == task.id }) else { throw StoreError.missing }
        var next = tasks; next[index] = task; try save(tasks: next, notes: notes, reminders: deliveredReminderKeys); tasks = next
    }

    func deleteTask(id: String) throws {
        guard tasks.contains(where: { $0.id == id }) else { throw StoreError.missing }
        let next = tasks.filter { $0.id != id }
        let prefix = "\(id)|"
        let reminders = deliveredReminderKeys.filter { !$0.hasPrefix(prefix) }
        try save(tasks: next, notes: notes, reminders: Set(reminders)); tasks = next; deliveredReminderKeys = Set(reminders)
    }

    func setCompleted(taskID: String, day: String?, completed: Bool) throws {
        guard var task = tasks.first(where: { $0.id == taskID }) else { throw StoreError.missing }
        let key = TaskSchedule.occurrenceKey(task, on: day)
        if completed { task.completedOccurrences.insert(key) } else { task.completedOccurrences.remove(key) }
        try updateTask(task)
    }

    func addNote(_ note: PlannerNote) throws {
        guard !note.title.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty || !note.body.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else {
            throw StoreError.invalid("笔记内容不能为空")
        }
        var next = notes; next.append(note); try save(tasks: tasks, notes: next, reminders: deliveredReminderKeys); notes = next
    }

    func updateNote(_ note: PlannerNote) throws {
        guard let index = notes.firstIndex(where: { $0.id == note.id }) else { throw StoreError.missing }
        guard !note.title.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty || !note.body.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else {
            throw StoreError.invalid("笔记内容不能为空")
        }
        var next = notes; next[index] = note; try save(tasks: tasks, notes: next, reminders: deliveredReminderKeys); notes = next
    }

    func deleteNote(id: String) throws {
        guard notes.contains(where: { $0.id == id }) else { throw StoreError.missing }
        let linked = tasks.contains(where: { $0.sourceNoteID == id })
        guard !linked else { throw StoreError.invalid("这篇笔记仍被待办引用，请先删除或修改关联待办") }
        let next = notes.filter { $0.id != id }; try save(tasks: tasks, notes: next, reminders: deliveredReminderKeys); notes = next
    }

    @discardableResult
    func createTask(from noteID: String, date: String?) throws -> PlannerTask {
        guard let note = notes.first(where: { $0.id == noteID }) else { throw StoreError.missing }
        let fallback = note.body.split(separator: "\n").first.map(String.init) ?? "来自笔记的待办"
        let title = note.title.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ? fallback : note.title
        let task = PlannerTask(title: title, details: note.body, startDate: date, sourceNoteID: note.id)
        try addTask(task); return task
    }

    func markRemindersDelivered(_ events: [ReminderEvent], now: Date = Date(), timeZone: TimeZone = .current) throws {
        guard !events.isEmpty else { return }
        let cutoff = TaskSchedule.ymd(Calendar(identifier: .gregorian).date(byAdding: .day, value: -14, to: now) ?? now, timeZone: timeZone)
        var next = Set(deliveredReminderKeys.filter { key in
            let parts = key.split(separator: "|", omittingEmptySubsequences: false)
            return parts.count < 2 || String(parts[1]) >= cutoff
        })
        events.forEach { next.insert($0.key) }
        try save(tasks: tasks, notes: notes, reminders: next); deliveredReminderKeys = next
    }

    private func save(tasks: [PlannerTask], notes: [PlannerNote], reminders: Set<String>) throws {
        guard !loadFailed else { throw StoreError.unreadable }
        let file = PlannerFile(schemaVersion: 1, tasks: tasks, notes: notes, deliveredReminderKeys: reminders)
        let encoder = JSONEncoder(); encoder.outputFormatting = [.prettyPrinted, .sortedKeys]
        let data = try encoder.encode(file)
        try FileManager.default.createDirectory(at: url.deletingLastPathComponent(), withIntermediateDirectories: true)
        try data.write(to: url, options: .atomic)
    }
}
