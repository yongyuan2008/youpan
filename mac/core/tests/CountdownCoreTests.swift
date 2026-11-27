import Foundation

@main struct Tests {
    static func main() throws {
        func check(_ condition: @autoclosure () -> Bool, _ message: String) { if !condition() { fatalError(message) } }
        func sizeIs(_ size: CGSize, _ width: CGFloat, _ height: CGFloat) -> Bool { size.width == width && size.height == height }
        func dt(_ value: String) -> Date { CountdownMath.date(value)! }
        func utc(_ value: String) -> Date { ISO8601DateFormatter().date(from: value)! }
        let utcZone = TimeZone(secondsFromGMT: 0)!

        // Countdown math and exact legacy data.json compatibility.
        check(CountdownMath.date("2024-02-29") != nil, "leap day")
        check(CountdownMath.date("2023-02-29") == nil, "invalid leap day")
        check(CountdownMath.date("2024-2-01") == nil, "strict date format")
        check(CountdownMath.display(.retirement, custom: .init(start: "2025-01-01"), today: dt("2024-12-31")).value == "1", "cross year")
        check(CountdownMath.display(.retirement, custom: .init(start: "2024-03-11"), today: dt("2024-03-09")).value == "2", "DST interval")
        let smallViewport = WindowGeometry.viewportSize(requested: CGSize(width: 760, height: 640), available: CGSize(width: 620, height: 510))
        check(sizeIs(smallViewport, 620, 510), "small screen viewport stays within visible content area")
        check(sizeIs(WindowGeometry.contentMinimum(available: CGSize(width: 620, height: 510)), 520, 440), "practical viewport minimum remains available on normal screens")
        check(sizeIs(WindowGeometry.contentMinimum(available: CGSize(width: 480, height: 400)), 480, 400), "viewport minimum shrinks for an exceptionally small screen")
        check(sizeIs(WindowGeometry.viewportSize(requested: CGSize(width: 1000, height: 640), available: CGSize(width: 820, height: 520)), 820, 520), "each viewport dimension clamps independently")
        check(sizeIs(WindowGeometry.persistedSize(actual: CGSize(width: 500, height: 410)), 520, 440), "exceptionally small viewport persists valid practical minimum")
        let fittedViewport = CGSize(width: 600, height: 540)
        let fittedScale = WindowGeometry.fitScale(viewport: fittedViewport)
        let fittedLayout = WindowGeometry.fittedLayoutSize(viewport: fittedViewport)
        check(fittedScale < 1 && fittedLayout.width >= 700 && fittedLayout.height >= 590, "small main window scales the complete baseline layout")
        check(fittedLayout.width * fittedScale <= fittedViewport.width + 0.01 && fittedLayout.height * fittedScale <= fittedViewport.height + 0.01, "scaled main content fits entirely inside 600 by 540 viewport")
        check(WindowGeometry.fitScale(viewport: CGSize(width: 760, height: 640)) == 1, "normal window keeps one-to-one rendering")
        let desktopFrame = DesktopPanelGeometry.cappedFrame(
            CGRect(origin: CGPoint(x: 950, y: 740), size: DesktopPanelGeometry.defaultSize),
            visibleFrames: [CGRect(x: 0, y: 0, width: 1000, height: 800)],
            fallback: CGRect(x: 0, y: 0, width: 1000, height: 800)
        )
        check(desktopFrame.maxX == 1000 && desktopFrame.maxY == 800, "desktop panel position caps to visible screen")
        let secondScreenFrame = DesktopPanelGeometry.cappedFrame(
            CGRect(origin: CGPoint(x: 1200, y: 40), size: DesktopPanelGeometry.defaultSize),
            visibleFrames: [CGRect(x: 0, y: 0, width: 1000, height: 800), CGRect(x: 1000, y: 0, width: 1000, height: 800)],
            fallback: CGRect(x: 0, y: 0, width: 1000, height: 800)
        )
        check(secondScreenFrame.minX == 1200, "desktop panel remains on its visible external screen")
        guard let testDirectory = ProcessInfo.processInfo.environment["RIZI_COUNTDOWN_TEST_DIRECTORY"], !testDirectory.isEmpty else {
            fatalError("RIZI_COUNTDOWN_TEST_DIRECTORY is required for isolated tests")
        }
        let root = URL(fileURLWithPath: testDirectory, isDirectory: true).appendingPathComponent("core-tests-\(UUID().uuidString)")
        defer { try? FileManager.default.removeItem(at: root) }
        let legacyURL = root.appendingPathComponent("data.json")
        let first = CountdownStore(url: legacyURL)
        check(!FileManager.default.fileExists(atPath: root.path), "no first launch write")
        check(first.selections["retirement"] == EventDates(start: "2046-01-10"), "retirement default retained")
        try FileManager.default.createDirectory(at: root, withIntermediateDirectories: true)
        let legacyJSON = #"{"selections":{"retirement":{"start":"2046-01-10"},"winter":{"start":"2027-01-18","end":"2027-02-18"}},"pinned":true}"#
        try Data(legacyJSON.utf8).write(to: legacyURL)
        let legacy = CountdownStore(url: legacyURL)
        check(legacy.pinned && legacy.selections["winter"]?.end == "2027-02-18", "v1 legacy file loads unchanged")
        try Data("invalid".utf8).write(to: legacyURL)
        let brokenLegacy = CountdownStore(url: legacyURL)
        check(brokenLegacy.loadWarning != nil, "legacy malformed warning")
        do { try brokenLegacy.setPinned(false); fatalError("must protect malformed legacy") } catch {}
        let legacyPreserved = try String(contentsOf: legacyURL, encoding: .utf8)
        check(legacyPreserved == "invalid", "legacy malformed preserved")

        // Independent appearance.json persistence, validation, names and corrupt-file protection.
        let appearanceURL = root.appendingPathComponent("appearance.json")
        let appearance = AppearanceStore(url: appearanceURL)
        check(appearance.configuration == .default, "appearance defaults without first-launch write")
        check(!FileManager.default.fileExists(atPath: appearanceURL.path), "appearance default does not write")
        let v22AppearanceURL = root.appendingPathComponent("appearance-v22.json")
        try Data(#"{"width":760,"height":640,"opacity":0.82,"names":{"retirement":"纪念日"}}"#.utf8).write(to: v22AppearanceURL)
        let v22Appearance = AppearanceStore(url: v22AppearanceURL)
        check(!v22Appearance.pinnedToDesktop && v22Appearance.name(for: .retirement) == "纪念日", "v2.2 appearance loads with desktop pin default off")
        try v22Appearance.setPinnedToDesktop(true)
        check(AppearanceStore(url: v22AppearanceURL).pinnedToDesktop, "desktop pin setting round trips")
        try appearance.updateSize(width: 1000, height: 780)
        try appearance.updateOpacity(0.73)
        try appearance.updateNames(["retirement": " 生日🎂 ", "winter": "雪季", "summer": "暑假"])
        check(appearance.name(for: .retirement) == "生日🎂" && appearance.name(for: .winter) == "雪季", "multiple names remain independent and trim whitespace")
        check(appearance.customName(for: .summer) == nil && appearance.name(for: .national) == EventKind.national.title, "default-equivalent and missing names fall back")
        let reopenedAppearance = AppearanceStore(url: appearanceURL)
        check(reopenedAppearance.width == 1000 && reopenedAppearance.height == 780 && reopenedAppearance.opacity == 0.73, "appearance size and opacity round trip")
        check(reopenedAppearance.name(for: .retirement) == "生日🎂" && reopenedAppearance.name(for: .winter) == "雪季", "Unicode names round trip")
        try reopenedAppearance.updateSize(width: 520, height: 440)
        try reopenedAppearance.updateOpacity(0.5)
        check(reopenedAppearance.width == 520 && reopenedAppearance.height == 440 && reopenedAppearance.opacity == 0.5, "minimum viewport boundaries accepted")
        try reopenedAppearance.resetAppearance()
        check(reopenedAppearance.width == 760 && reopenedAppearance.height == 640 && reopenedAppearance.opacity == 1 && !reopenedAppearance.pinnedToDesktop, "default appearance reset")
        check(reopenedAppearance.name(for: .retirement) == "生日🎂", "appearance reset preserves names")

        let appearanceBeforeInvalid = reopenedAppearance.configuration
        do { try reopenedAppearance.updateNames(["retirement": " \n "]); fatalError("blank name must fail") } catch {}
        do { try reopenedAppearance.updateNames(["retirement": String(repeating: "名", count: 25)]); fatalError("overlong name must fail") } catch {}
        do { try reopenedAppearance.updateSize(width: .nan, height: 640); fatalError("NaN size must fail") } catch {}
        do { try reopenedAppearance.updateSize(width: 519, height: 440); fatalError("undersized width must fail") } catch {}
        do { try reopenedAppearance.updateSize(width: 760, height: 4097); fatalError("oversized height must fail") } catch {}
        do { try reopenedAppearance.updateOpacity(.infinity); fatalError("infinite opacity must fail") } catch {}
        do { try reopenedAppearance.updateOpacity(0.49); fatalError("opacity below boundary must fail") } catch {}
        check(reopenedAppearance.configuration == appearanceBeforeInvalid, "invalid appearance changes are atomic")
        try reopenedAppearance.resetNames()
        check(reopenedAppearance.configuration.names.isEmpty && reopenedAppearance.name(for: .retirement) == EventKind.retirement.title, "default name reset preserves fallback")

        let defaultDisplay = CountdownMath.display(.retirement, custom: .init(start: "2026-12-05"), today: dt("2026-11-25"))
        let renamedDisplay = CountdownMath.display(.retirement, custom: .init(start: "2026-12-05"), today: dt("2026-11-25"), customName: "生日")
        check(defaultDisplay.value == renamedDisplay.value && defaultDisplay.dateLine == renamedDisplay.dateLine, "rename does not alter date or day count")
        check(renamedDisplay.caption == "距生日" && !renamedDisplay.caption.contains("退休"), "renamed future caption is generic")
        let activeRenamed = CountdownMath.display(.winter, custom: .init(start: "2026-11-20", end: "2026-11-30"), today: dt("2026-11-25"), customName: "驻场")
        let endedRenamed = CountdownMath.display(.winter, custom: .init(start: "2026-11-20", end: "2026-11-22"), today: dt("2026-11-25"), customName: "驻场")
        check(activeRenamed.caption == "进行中" && endedRenamed.caption == "已结束", "renamed range uses neutral captions")
        check(CountdownMath.display(.retirement, custom: .init(start: "2026-11-25"), today: dt("2026-11-25"), customName: "生日").caption == "今天", "renamed same-day caption is neutral")

        let brokenAppearanceURL = root.appendingPathComponent("broken-appearance.json")
        let brokenAppearanceText = #"{"width":760,"height":640,"opacity":0.25,"names":{}}"#
        try Data(brokenAppearanceText.utf8).write(to: brokenAppearanceURL)
        let brokenAppearance = AppearanceStore(url: brokenAppearanceURL)
        check(brokenAppearance.loadWarning != nil, "invalid appearance warning")
        do { try brokenAppearance.updateOpacity(0.8); fatalError("must protect malformed appearance") } catch {}
        let brokenAppearancePreserved = try String(contentsOf: brokenAppearanceURL, encoding: .utf8)
        check(brokenAppearancePreserved == brokenAppearanceText, "invalid appearance file preserved")

        // Strict task validation and planner persistence.
        check(PlannerValidation.validTime("09:05"), "valid reminder time")
        check(!PlannerValidation.validTime("+1:30") && !PlannerValidation.validTime("9:05") && !PlannerValidation.validTime("２１:３０"), "ASCII HH:mm only")
        check(PlannerValidation.validate(PlannerTask(title: "x", reminderTime: "09:00")) != nil, "unscheduled reminder rejected")
        let plannerURL = root.appendingPathComponent("planner.json")
        let store = PlannerStore(url: plannerURL)
        let task = PlannerTask(id: "t1", title: "晨会", details: "准备材料", startDate: "2026-11-23", reminderTime: "09:00", recurrence: .weekdays, isPriority: true, createdAt: "2026-01-01T00:00:00.000Z")
        let unscheduledEdit = PlannerTask(id: "u1", title: "未安排")
        check(TaskEditorDefaults.initialDate(existing: unscheduledEdit, suggestedDate: "2026-11-25") == nil, "editing unscheduled task preserves no date")
        check(TaskEditorDefaults.initialDate(existing: task, suggestedDate: "2026-11-25") == "2026-11-23", "editing scheduled task preserves own date")
        check(TaskEditorDefaults.initialDate(existing: nil, suggestedDate: "2026-11-25") == "2026-11-25", "new task accepts selected date seed")
        try store.addTask(task)
        let reopened = PlannerStore(url: plannerURL)
        check(reopened.tasks == [task], "task round trip")
        try Data("{".utf8).write(to: plannerURL)
        let brokenPlanner = PlannerStore(url: plannerURL)
        check(brokenPlanner.loadWarning != nil, "planner malformed warning")
        do { try brokenPlanner.addTask(PlannerTask(title: "do not write")); fatalError("must protect malformed planner") } catch {}
        let plannerPreserved = try String(contentsOf: plannerURL, encoding: .utf8)
        check(plannerPreserved == "{", "planner malformed preserved")

        // Recurrence and per-occurrence completion.
        let daily = PlannerTask(id: "daily", title: "每日复盘", startDate: "2026-11-23", recurrence: .daily)
        let weekdays = task
        let weekly = PlannerTask(id: "weekly", title: "周会", startDate: "2026-11-23", recurrence: .weekly)
        check(!TaskSchedule.occurs(daily, on: "2026-11-22", timeZone: utcZone), "recurrence never precedes start")
        check(TaskSchedule.occurs(daily, on: "2026-11-24", timeZone: utcZone), "daily occurrence")
        check(TaskSchedule.occurs(weekdays, on: "2026-11-27", timeZone: utcZone) && !TaskSchedule.occurs(weekdays, on: "2026-11-28", timeZone: utcZone), "weekday Monday-Friday")
        check(TaskSchedule.occurs(weekly, on: "2026-11-30", timeZone: utcZone) && !TaskSchedule.occurs(weekly, on: "2026-12-01", timeZone: utcZone), "weekly recurrence")
        let completionURL = root.appendingPathComponent("completion.json")
        let completionStore = PlannerStore(url: completionURL); try completionStore.addTask(daily)
        try completionStore.setCompleted(taskID: daily.id, day: "2026-11-24", completed: true)
        let completedTask = completionStore.tasks[0]
        check(TaskSchedule.isCompleted(completedTask, on: "2026-11-24") && !TaskSchedule.isCompleted(completedTask, on: "2026-11-25"), "completion is occurrence-specific")

        // Same-local-day catch-up, future suppression, completion and persisted dedup.
        let reminderTask = PlannerTask(id: "r1", title: "早会", startDate: "2026-11-25", reminderTime: "09:00")
        let now = utc("2026-11-25T14:00:00Z")
        let due = ReminderLogic.due(tasks: [reminderTask], delivered: [], now: now, timeZone: utcZone)
        check(due.count == 1 && due[0].occurrence == "2026-11-25", "same-day catch-up")
        let simultaneous = PlannerTask(id: "r2", title: "另一项", startDate: "2026-11-25", reminderTime: "09:00")
        let simultaneousDue = ReminderLogic.due(tasks: [reminderTask, simultaneous], delivered: [], now: now, timeZone: utcZone)
        check(simultaneousDue.count == 2, "simultaneous reminders grouped in one due batch")
        let mergedUnread = ReminderLogic.mergingUnread(due, with: simultaneousDue)
        check(mergedUnread.count == 2 && mergedUnread[0].key == due[0].key, "new real reminders merge into the unread panel without duplicates")
        check(ReminderLogic.due(tasks: [reminderTask], delivered: [], now: utc("2026-11-25T08:59:00Z"), timeZone: utcZone).isEmpty, "future reminder suppressed")
        check(ReminderLogic.due(tasks: [reminderTask], delivered: Set([due[0].key]), now: now, timeZone: utcZone).isEmpty, "dedup suppresses")
        var doneReminder = reminderTask; doneReminder.completedOccurrences.insert("2026-11-25")
        check(ReminderLogic.due(tasks: [doneReminder], delivered: [], now: now, timeZone: utcZone).isEmpty, "completed reminder suppressed")
        let yesterday = PlannerTask(id: "old", title: "昨天", startDate: "2026-11-24", reminderTime: "09:00")
        check(ReminderLogic.due(tasks: [yesterday], delivered: [], now: now, timeZone: utcZone).isEmpty, "no previous-day backlog")
        let newYork = TimeZone(identifier: "America/New_York")!, shanghai = TimeZone(identifier: "Asia/Shanghai")!
        let zoneInstant = utc("2026-11-25T01:00:00Z")
        let nyTask = PlannerTask(id: "ny", title: "纽约本地日", startDate: "2026-11-24", reminderTime: "19:00")
        let shTask = PlannerTask(id: "sh", title: "上海本地日", startDate: "2026-11-25", reminderTime: "08:00")
        check(ReminderLogic.due(tasks: [nyTask], delivered: [], now: zoneInstant, timeZone: newYork).first?.occurrence == "2026-11-24", "New York local date")
        check(ReminderLogic.due(tasks: [shTask], delivered: [], now: zoneInstant, timeZone: shanghai).first?.occurrence == "2026-11-25", "Shanghai local date")
        let dstTask = PlannerTask(id: "dst", title: "夏令时", startDate: "2026-03-08", reminderTime: "02:30")
        check(ReminderLogic.due(tasks: [dstTask], delivered: [], now: utc("2026-03-08T08:00:00Z"), timeZone: newYork).count == 1, "DST missing hour catches up same local day")
        let reminderStoreURL = root.appendingPathComponent("reminders.json")
        let reminderStore = PlannerStore(url: reminderStoreURL); try reminderStore.addTask(reminderTask); try reminderStore.markRemindersDelivered(due, now: now, timeZone: utcZone)
        check(PlannerStore(url: reminderStoreURL).deliveredReminderKeys.contains(due[0].key), "dedup persists")

        // Monday-first fixed 6-week calendar grid.
        let grid = CalendarGrid(containing: utc("2026-11-25T12:00:00Z"), timeZone: utcZone)
        check(grid.days.count == 42, "calendar has six weeks")
        check(TaskSchedule.ymd(grid.days[0], timeZone: utcZone) == "2026-10-26", "calendar starts Monday")
        var utcCalendar = Calendar(identifier: .gregorian); utcCalendar.timeZone = utcZone
        check(utcCalendar.component(.weekday, from: grid.days[0]) == 2, "first grid day is Monday")

        // Pure seven-day agenda selector crosses month/year without copying unscheduled records.
        var agendaDaily = PlannerTask(id: "a-daily", title: "跨年复盘", details: "查看详细进度", startDate: "2026-12-29", recurrence: .daily)
        agendaDaily.completedOccurrences.insert("2026-12-31")
        let agendaWeekdays = PlannerTask(id: "a-work", title: "工作日签到", startDate: "2026-12-29", recurrence: .weekdays)
        let agendaWeekly = PlannerTask(id: "a-weekly", title: "每周事项", startDate: "2026-12-29", recurrence: .weekly)
        let agendaOnce = PlannerTask(id: "a-once", title: "新年任务", startDate: "2027-01-02")
        let agendaUnscheduled = PlannerTask(id: "a-none", title: "未安排不应出现")
        let agendaGroups = AgendaSelector.upcoming(tasks: [agendaDaily, agendaWeekdays, agendaWeekly, agendaOnce, agendaUnscheduled], notes: [], startDay: "2026-12-29", dayCount: 7, timeZone: utcZone)
        check(agendaGroups.first?.day == "2026-12-29" && agendaGroups.last?.day == "2027-01-04", "agenda crosses month and year in inclusive seven-day window")
        check(!agendaGroups.flatMap(\.occurrences).contains(where: { $0.task.id == "a-none" }), "agenda excludes unscheduled tasks")
        check(agendaGroups.first(where: { $0.day == "2027-01-02" })?.occurrences.contains(where: { $0.task.id == "a-work" }) == false, "agenda respects weekday recurrence on weekend")
        check(agendaGroups.first(where: { $0.day == "2026-12-31" })?.occurrences.first(where: { $0.task.id == "a-daily" })?.completed == true && agendaGroups.first(where: { $0.day == "2027-01-01" })?.occurrences.first(where: { $0.task.id == "a-daily" })?.completed == false, "agenda completion stays occurrence-specific")
        check(AgendaSelector.upcoming(tasks: [agendaDaily], notes: [], startDay: "2026-12-29", dayCount: 7, query: "详细进度", timeZone: utcZone).count == 7, "agenda query matches task details")
        let sourceSearchNote = PlannerNote(id: "search-note", title: "灵感来源", body: "", createdAt: "2026-01-01T00:00:00.000Z", updatedAt: "2026-01-01T00:00:00.000Z")
        var sourcedTask = agendaOnce; sourcedTask.sourceNoteID = sourceSearchNote.id
        check(AgendaSelector.matches(sourcedTask, query: "灵感", notes: [sourceSearchNote]), "agenda query matches source note title")

        // Note remains after linked task creation; sourceNoteID is durable.
        let linkURL = root.appendingPathComponent("links.json")
        let linkStore = PlannerStore(url: linkURL)
        let note = PlannerNote(id: "n1", title: "项目构想", body: "整理研究问题", createdAt: "2026-01-01T00:00:00.000Z", updatedAt: "2026-01-01T00:00:00.000Z")
        try linkStore.addNote(note)
        let linked = try linkStore.createTask(from: note.id, date: "2026-11-25")
        check(linkStore.notes == [note] && linked.sourceNoteID == note.id, "note retained and linked")
        let linkedReopen = PlannerStore(url: linkURL)
        check(linkedReopen.tasks.first?.sourceNoteID == "n1" && linkedReopen.notes.count == 1, "note link round trip")
        do { try linkedReopen.deleteNote(id: "n1"); fatalError("linked note must be protected") } catch {}

        // Central quit guard saves dirty note drafts and blocks termination when saving fails.
        let saveGuard = UnsavedNoteGuard(); var saveCount = 0
        saveGuard.register { saveCount += 1; return true }; saveGuard.setDirty(true)
        check(saveGuard.attemptSave() && saveCount == 1 && !saveGuard.isDirty && saveGuard.editorTitle == "笔记", "dirty quit guard saves once")
        saveGuard.register(editorTitle: "倒计时名称", editorObject: "修改后的倒计时名称") { true }
        saveGuard.setDirty(true)
        check(saveGuard.editorTitle == "倒计时名称" && saveGuard.editorObject == "修改后的倒计时名称", "quit guard provides the active editor prompt")
        saveGuard.register { false }; saveGuard.setDirty(true)
        check(!saveGuard.attemptSave() && saveGuard.isDirty, "failed quit save remains dirty and blocks termination")
        saveGuard.discard(); check(!saveGuard.isDirty, "explicit discard clears quit guard")

        print("All deterministic core checks passed")
    }
}
