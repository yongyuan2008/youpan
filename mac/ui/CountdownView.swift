import SwiftUI
import AppKit
import Combine

private let ink = Color(red: 0.11, green: 0.16, blue: 0.24)
private let secondaryInk = Color(red: 0.33, green: 0.39, blue: 0.48)
private let paper = Color(red: 0.955, green: 0.968, blue: 0.984)
private let macBlue = Color(red: 0.04, green: 0.35, blue: 0.78)
private let dividerInk = Color(red: 0.87, green: 0.90, blue: 0.94)
private let sidebarPaper = Color(red: 0.92, green: 0.94, blue: 0.96)

private enum AppSection: String, CaseIterable, Identifiable {
    case countdown = "倒计时", today = "今天", calendar = "日历", notes = "笔记"
    var id: String { rawValue }
    var icon: String {
        switch self { case .countdown: return "hourglass"; case .today: return "checklist"; case .calendar: return "calendar"; case .notes: return "note.text" }
    }
    var shortcut: KeyEquivalent {
        switch self { case .countdown: return "1"; case .today: return "2"; case .calendar: return "3"; case .notes: return "4" }
    }
}

struct CountdownView: View {
    @ObservedObject var store: CountdownStore
    @ObservedObject var planner: PlannerStore
    @ObservedObject var appearance: AppearanceStore
    @State(wrappedValue: AppSection.countdown) private var selection: AppSection
    @State(wrappedValue: 165.0 / 760.0) private var sidebarFraction: CGFloat
    @State(wrappedValue: false) private var showingSettings: Bool
    @State(wrappedValue: nil) private var configurationWarning: String?
    var body: some View {
        GeometryReader { geometry in
            let width = geometry.size.width
            let sidebarWidth = constrainedSidebarWidth(width * sidebarFraction, in: width)
            HStack(spacing: 0) {
                sidebar(compact: sidebarWidth < 116)
                    .frame(width: sidebarWidth)
                sidebarDivider(width: width)
                Group {
                    switch selection {
                    case .countdown: CountdownDashboard(store: store, appearance: appearance)
                    case .today: TodayView(planner: planner)
                    case .calendar: PlannerCalendarView(planner: planner, countdown: store, appearance: appearance)
                    case .notes: NotesView(planner: planner)
                    }
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity)
                .background(paper)
            }.coordinateSpace(name: "mainSplit")
        }.tint(macBlue).preferredColorScheme(.light)
        .sheet(isPresented: $showingSettings) { AppearanceSettingsView(appearance: appearance) }
        .onAppear {
            if configurationWarning == nil {
                configurationWarning = [planner.loadWarning, appearance.loadWarning].compactMap { $0 }.joined(separator: "\n\n")
                if configurationWarning?.isEmpty == true { configurationWarning = nil }
            }
        }
        .alert("配置无法读取", isPresented: Binding(get: { configurationWarning != nil }, set: { if !$0 { configurationWarning = nil } })) {
            Button("好", role: .cancel) {}
        } message: { Text(configurationWarning ?? "") }
    }

    private func constrainedSidebarWidth(_ proposed: CGFloat, in width: CGFloat) -> CGFloat {
        min(max(52, proposed), max(52, min(240, width - 490)))
    }

    private func sidebarDivider(width: CGFloat) -> some View {
        Rectangle()
            .fill(sidebarPaper)
            .overlay { Rectangle().fill(ink.opacity(0.13)).frame(width: 1) }
            .frame(width: 7)
            .contentShape(Rectangle())
            .onHover { hovering in
                if hovering { NSCursor.resizeLeftRight.set() }
                else { NSCursor.arrow.set() }
            }
            .gesture(DragGesture(minimumDistance: 0, coordinateSpace: .named("mainSplit"))
                .onChanged { value in
                    sidebarFraction = constrainedSidebarWidth(value.location.x - 3.5, in: width) / max(1, width)
                })
            .accessibilityLabel("侧栏宽度")
            .accessibilityAdjustableAction { direction in
                let change: CGFloat
                switch direction {
                case .increment: change = 16
                case .decrement: change = -16
                @unknown default: return
                }
                sidebarFraction = constrainedSidebarWidth(width * sidebarFraction + change, in: width) / max(1, width)
            }
    }

    private func sidebar(compact: Bool) -> some View {
        VStack(spacing: 0) {
            VStack(spacing: 3) {
                ForEach(AppSection.allCases) { item in
                    sidebarFocusStyle(Button { selection = item } label: {
                        sidebarLabel(item.rawValue, icon: item.icon, compact: compact)
                            .foregroundColor(selection == item ? Color(red: 0.0, green: 0.31, blue: 0.72) : ink)
                            .background(selection == item ? macBlue.opacity(0.11) : Color.clear)
                            .cornerRadius(8)
                    })
                    .buttonStyle(.plain)
                    .keyboardShortcut(item.shortcut, modifiers: .command)
                    .help(item.rawValue)
                    .accessibilityLabel(item.rawValue)
                    .accessibilityAddTraits(selection == item ? .isSelected : [])
                }
            }
            .padding(.horizontal, 8)
            .padding(.top, 10)
            Spacer(minLength: 0)
            Divider()
            Button { showingSettings = true } label: {
                sidebarLabel("设置", icon: "gearshape", compact: compact)
            }
            .buttonStyle(.plain)
            .padding(8)
            .help("设置")
            .accessibilityLabel("打开外观与倒计时名称设置")
        }
        .background(sidebarPaper)
    }

    @ViewBuilder private func sidebarFocusStyle<Content: View>(_ content: Content) -> some View {
        if #available(macOS 14, *) { content.focusEffectDisabled() }
        else { content }
    }

    private func sidebarLabel(_ title: String, icon: String, compact: Bool) -> some View {
        HStack(spacing: 8) {
            Image(systemName: icon).frame(width: 20)
            if !compact { Text(title).lineLimit(1) }
        }
        .padding(.horizontal, compact ? 0 : 9)
        .frame(maxWidth: .infinity, minHeight: 32, alignment: compact ? .center : .leading)
        .contentShape(Rectangle())
    }
}

private struct CountdownDashboard: View {
    @ObservedObject var store: CountdownStore
    @ObservedObject var appearance: AppearanceStore
    @State(wrappedValue: Date()) private var now: Date
    @State(wrappedValue: nil) private var editing: EventKind?
    @State(wrappedValue: nil) private var errorText: String?
    @State(wrappedValue: false) private var showedLoadWarning: Bool
    private let timer = Timer.publish(every: 60, on: .main, in: .common).autoconnect()
    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 14) {
                HStack(alignment: .top) {
                    VStack(alignment: .leading, spacing: 3) {
                        Text("倒计时").font(.system(size: 25, weight: .bold, design: .rounded))
                        Text("今天 · \(todayText)").font(.system(size: 12)).foregroundColor(secondaryInk)
                    }
                    Spacer()
                    Toggle("窗口置顶", isOn: Binding(get: { store.pinned }, set: { value in
                        do { try store.setPinned(value) } catch { errorText = "无法保存置顶设置：\(error.localizedDescription)" }
                    })).toggleStyle(.switch).controlSize(.mini).font(.system(size: 12)).padding(.top, 5)
                }
                retirementCard
                Text("其他倒计时").font(.system(size: 14, weight: .semibold)).padding(.top, 1)
                otherCountdownRows
            }.padding(24)
        }.foregroundColor(ink)
        .onReceive(timer) { now = $0 }
        .onReceive(NotificationCenter.default.publisher(for: NSApplication.didBecomeActiveNotification)) { _ in now = Date() }
        .onReceive(NSWorkspace.shared.notificationCenter.publisher(for: NSWorkspace.didWakeNotification)) { _ in now = Date() }
        .onReceive(NotificationCenter.default.publisher(for: Notification.Name.NSSystemTimeZoneDidChange)) { _ in now = Date() }
        .onReceive(NotificationCenter.default.publisher(for: Notification.Name.NSCalendarDayChanged)) { _ in now = Date() }
        .onAppear { if !showedLoadWarning, let warning = store.loadWarning { errorText = warning; showedLoadWarning = true } }
        .sheet(item: $editing) { EventEditor(kind: $0, name: appearance.name(for: $0), store: store, today: now) }
        .alert("提示", isPresented: Binding(get: { errorText != nil }, set: { if !$0 { errorText = nil } })) { Button("好", role: .cancel) {} } message: { Text(errorText ?? "") }
    }
    private var todayText: String { let f = DateFormatter(); f.locale = Locale(identifier: "zh_CN"); f.calendar = Calendar(identifier: .gregorian); f.timeZone = .current; f.dateFormat = "yyyy年M月d日 EEEE"; return f.string(from: now) }
    private func display(_ kind: EventKind) -> CountdownDisplay {
        CountdownMath.display(kind, custom: store.selections[kind.rawValue], today: now, customName: appearance.customName(for: kind))
    }
    private var retirementCard: some View {
        let item = display(.retirement)
        let name = appearance.name(for: .retirement)
        let headline: String = {
            guard appearance.customName(for: .retirement) != nil else {
                return item.caption == "距退休" ? "距我的退休" : (item.caption.hasPrefix("设置") ? "我的退休" : item.caption)
            }
            if item.caption.hasPrefix("设置") { return "设置\(name)日期" }
            if item.caption == "距\(name)" { return item.caption }
            return "\(name) · \(item.caption)"
        }()
        return Button { editing = .retirement } label: {
            HStack(spacing: 0) {
                VStack(alignment: .leading, spacing: 6) {
                    Text(headline).font(.system(size: 12, weight: .medium)).foregroundColor(.white.opacity(0.78))
                    HStack(alignment: .firstTextBaseline, spacing: 4) {
                        Text(item.value).font(.system(size: item.value.count > 4 ? 28 : 41, weight: .bold, design: .rounded))
                        if !item.unit.isEmpty { Text(item.unit).font(.system(size: 14, weight: .medium)) }
                    }
                    if !item.dateLine.isEmpty { Text(item.dateLine).font(.system(size: 11)).foregroundColor(.white.opacity(0.75)) }
                }.frame(maxWidth: .infinity, alignment: .leading)
                SunriseScene().frame(width: 140, height: 92).accessibilityHidden(true)
            }.padding(.horizontal, 20).padding(.vertical, 16).frame(maxWidth: .infinity, minHeight: 126)
                .foregroundColor(.white).background(LinearGradient(colors: [Color(red: 0.137, green: 0.278, blue: 0.651), Color(red: 0.278, green: 0.486, blue: 0.878)], startPoint: .leading, endPoint: .trailing)).cornerRadius(17).contentShape(Rectangle())
        }.buttonStyle(.plain).accessibilityLabel("\(name)，\(item.caption)，\(item.value)\(item.unit)，\(item.dateLine)")
    }
    private func icon(_ kind: EventKind) -> String { switch kind { case .winter: return "snowflake"; case .summer: return "sun.max"; case .labor: return "leaf"; case .national: return "flag"; case .retirement: return "calendar" } }
    private var otherCountdownRows: some View {
        VStack(spacing: 0) {
            holidayRow(.winter); Divider().padding(.leading, 59)
            holidayRow(.summer); Divider().padding(.leading, 59)
            holidayRow(.labor); Divider().padding(.leading, 59)
            holidayRow(.national)
        }.background(Color.white).cornerRadius(14).overlay(RoundedRectangle(cornerRadius: 14).stroke(Color.black.opacity(0.055)))
    }
    private func holidayRow(_ kind: EventKind) -> some View {
        let item = display(kind)
        let name = appearance.name(for: kind)
        return Button { editing = kind } label: {
            HStack(spacing: 11) {
                Image(systemName: icon(kind)).font(.system(size: 14, weight: .medium)).foregroundColor(macBlue).frame(width: 35, height: 35).background(macBlue.opacity(0.1)).cornerRadius(10)
                VStack(alignment: .leading, spacing: 4) { Text(name).font(.system(size: 14, weight: .semibold)); if !item.dateLine.isEmpty { Text(item.dateLine).font(.system(size: 11)).foregroundColor(secondaryInk) } }
                Spacer()
                VStack(alignment: .trailing, spacing: 3) { HStack(alignment: .firstTextBaseline, spacing: 2) { Text(item.value).font(.system(size: 19, weight: .semibold, design: .rounded)); Text(item.unit).font(.system(size: 11)) }; if !item.caption.hasPrefix("设置") { Text(item.caption).font(.system(size: 10)).foregroundColor(secondaryInk) } }
            }.padding(.horizontal, 15).frame(height: 61).contentShape(Rectangle())
        }.buttonStyle(.plain).accessibilityLabel("\(name)，\(item.caption)，\(item.value)\(item.unit)，\(item.dateLine)")
    }
}

private enum TodayScope: String, CaseIterable, Identifiable {
    case today = "今日", nextSeven = "未来7天"
    var id: String { rawValue }
}

private struct TodayView: View {
    @ObservedObject var planner: PlannerStore
    @State(wrappedValue: Date()) private var now: Date
    @State(wrappedValue: "") private var quickTitle: String
    @State(wrappedValue: nil) private var editorTask: PlannerTask?
    @State(wrappedValue: false) private var showingNew: Bool
    @State(wrappedValue: false) private var completedExpanded: Bool
    @State(wrappedValue: nil) private var errorText: String?
    @State(wrappedValue: "") private var query: String
    @State(wrappedValue: TodayScope.today) private var scope: TodayScope
    private let timer = Timer.publish(every: 60, on: .main, in: .common).autoconnect()
    private var day: String { TaskSchedule.ymd(now) }
    var body: some View {
        let sections = TaskSchedule.todaySections(tasks: planner.tasks, day: day)
        let matches: (PlannerTask) -> Bool = { AgendaSelector.matches($0, query: query, notes: planner.notes) }
        let upcoming = AgendaSelector.upcoming(tasks: planner.tasks, notes: planner.notes, startDay: day, dayCount: 7, query: query)
        VStack(alignment: .leading, spacing: 0) {
            PageHeader(title: "今天", subtitle: dateHeading) { Button { showingNew = true } label: { Label("新建待办", systemImage: "plus") }.accessibilityLabel("新建待办") }
            ScrollView {
                VStack(alignment: .leading, spacing: 18) {
                    Picker("查看范围", selection: $scope) { ForEach(TodayScope.allCases) { Text($0.rawValue).tag($0) } }.pickerStyle(.segmented).accessibilityLabel("待办查看范围")
                    TextField("搜索标题、详情或来源笔记", text: $query).textFieldStyle(.roundedBorder).accessibilityLabel("搜索待办")
                    if scope == .today {
                        HStack { TextField("快速添加待办", text: $quickTitle).textFieldStyle(.roundedBorder).onSubmit(addQuick); Button(action: addQuick) { Image(systemName: "plus.circle.fill") }.buttonStyle(.plain).font(.title2).accessibilityLabel("添加快速待办") }
                        let overdue = sections.overdue.filter(matches), current = sections.current.filter(matches), unscheduled = sections.unscheduled.filter(matches), completed = sections.completed.filter(matches)
                        if !overdue.isEmpty { TaskSection(title: "已逾期", tasks: overdue, dayProvider: { $0.startDate }, planner: planner, editorTask: $editorTask, errorText: $errorText, accent: .red) }
                        TaskSection(title: "今天", tasks: current, dayProvider: { _ in day }, planner: planner, editorTask: $editorTask, errorText: $errorText)
                        if !unscheduled.isEmpty { TaskSection(title: "未安排", tasks: unscheduled, dayProvider: { _ in nil }, planner: planner, editorTask: $editorTask, errorText: $errorText) }
                        DisclosureGroup(isExpanded: $completedExpanded) { TaskSection(title: "", tasks: completed, dayProvider: { $0.startDate == nil ? nil : day }, planner: planner, editorTask: $editorTask, errorText: $errorText, hideHeader: true) } label: { Text("已完成（\(completed.count)）").font(.headline) }
                    } else if upcoming.isEmpty {
                        Text(query.isEmpty ? "未来 7 天没有已安排待办" : "未来 7 天没有匹配的待办").foregroundColor(secondaryInk).frame(maxWidth: .infinity).padding(.vertical, 30)
                    } else {
                        ForEach(upcoming) { group in
                            VStack(alignment: .leading, spacing: 8) {
                                Text(agendaHeading(group.day)).font(.headline).foregroundColor(macBlue)
                                ForEach(group.occurrences) { occurrence in
                                    TaskRow(task: occurrence.task, occurrenceDay: occurrence.day, planner: planner, edit: { editorTask = occurrence.task }, errorText: $errorText)
                                }
                            }
                        }
                    }
                    ReminderFooter()
                }.padding(24)
            }
        }.onReceive(timer) { now = $0 }.onReceive(NotificationCenter.default.publisher(for: Notification.Name.NSCalendarDayChanged)) { _ in now = Date() }
        .sheet(isPresented: $showingNew) { TaskEditor(planner: planner, initialDate: day) }
        .sheet(item: $editorTask) { TaskEditor(planner: planner, task: $0, initialDate: day) }
        .alert("提示", isPresented: Binding(get: { errorText != nil }, set: { if !$0 { errorText = nil } })) { Button("好", role: .cancel) {} } message: { Text(errorText ?? "") }
    }
    private var dateHeading: String { let f = DateFormatter(); f.locale = Locale(identifier: "zh_CN"); f.dateFormat = "M月d日 EEEE"; return f.string(from: now) }
    private func agendaHeading(_ value: String) -> String { guard let date = CountdownMath.date(value) else { return value }; let f = DateFormatter(); f.locale = Locale(identifier: "zh_CN"); f.dateFormat = "M月d日 EEEE"; return f.string(from: date) }
    private func addQuick() {
        let value = quickTitle.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !value.isEmpty else { return }
        do {
            try planner.addTask(PlannerTask(title: value, startDate: day))
            quickTitle = ""
            query = ""
        } catch { errorText = error.localizedDescription }
    }
}

private struct TaskSection: View {
    let title: String, tasks: [PlannerTask], dayProvider: (PlannerTask) -> String?
    @ObservedObject var planner: PlannerStore
    @Binding var editorTask: PlannerTask?
    @Binding var errorText: String?
    var accent: Color = macBlue; var hideHeader = false
    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            if !hideHeader { Text(title).font(.headline).foregroundColor(accent) }
            if tasks.isEmpty && !hideHeader { Text("暂无待办").font(.subheadline).foregroundColor(secondaryInk).padding(.vertical, 5) }
            ForEach(tasks) { task in TaskRow(task: task, occurrenceDay: dayProvider(task), planner: planner, edit: { editorTask = task }, errorText: $errorText) }
        }
    }
}

private struct TaskRow: View {
    let task: PlannerTask, occurrenceDay: String?
    @ObservedObject var planner: PlannerStore
    let edit: () -> Void
    @Binding var errorText: String?
    private var done: Bool { TaskSchedule.isCompleted(task, on: occurrenceDay) }
    var body: some View {
        HStack(spacing: 10) {
            Button { do { try planner.setCompleted(taskID: task.id, day: occurrenceDay, completed: !done) } catch { errorText = error.localizedDescription } } label: { Image(systemName: done ? "checkmark.circle.fill" : "circle").foregroundColor(done ? .green : secondaryInk).font(.title3) }.buttonStyle(.plain).accessibilityLabel(done ? "标记未完成：\(task.title)" : "完成：\(task.title)")
            Button(action: edit) {
                HStack {
                    VStack(alignment: .leading, spacing: 3) {
                        HStack(spacing: 5) { if task.isPriority { Image(systemName: "star.fill").foregroundColor(.orange).font(.caption) }; Text(task.title).strikethrough(done).font(.system(size: 14, weight: .medium)).lineLimit(1) }
                        HStack(spacing: 8) { if let time = task.reminderTime { Label(time, systemImage: "bell").font(.caption).foregroundColor(secondaryInk) }; if task.recurrence != .none { Text(task.recurrence == .weekdays ? "工作日（周一至周五）" : task.recurrence.title).font(.caption).foregroundColor(secondaryInk) }; if task.sourceNoteID != nil { Label("来自笔记", systemImage: "link").font(.caption).foregroundColor(secondaryInk) } }
                    }
                    Spacer(); Image(systemName: "chevron.right").font(.caption).foregroundColor(secondaryInk.opacity(0.6))
                }.contentShape(Rectangle())
            }.buttonStyle(.plain).accessibilityLabel("编辑待办：\(task.title)")
        }.padding(11).background(Color.white).cornerRadius(11)
    }
}

private struct PlannerCalendarView: View {
    private struct NewTaskDay: Identifiable {
        let id: String
    }

    @ObservedObject var planner: PlannerStore
    @ObservedObject var countdown: CountdownStore
    @ObservedObject var appearance: AppearanceStore
    @State(wrappedValue: Date()) private var visibleMonth: Date
    @State(wrappedValue: Date()) private var selectedDay: Date
    @State(wrappedValue: nil) private var editorTask: PlannerTask?
    @State(wrappedValue: nil) private var newTaskDay: NewTaskDay?
    @State(wrappedValue: nil) private var errorText: String?
    private let weekdays = ["周一", "周二", "周三", "周四", "周五", "周六", "周日"]
    var body: some View {
        let grid = CalendarGrid(containing: visibleMonth), key = TaskSchedule.ymd(selectedDay), dayTasks = planner.tasks.filter { TaskSchedule.occurs($0, on: key) }
        VStack(alignment: .leading, spacing: 0) {
            PageHeader(title: "日历", subtitle: "双击日期添加待办") {
                Button("今天") { visibleMonth = Date(); selectedDay = Date() }.accessibilityLabel("回到今天")
            }
            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    VStack(spacing: 8) {
                        HStack {
                            Text(monthTitle).font(.system(size: 19, weight: .semibold)).monospacedDigit()
                            Spacer()
                            monthButton(-1, icon: "chevron.left", label: "上个月")
                            monthButton(1, icon: "chevron.right", label: "下个月")
                        }.padding(.bottom, 3)
                        HStack(spacing: 4) {
                            ForEach(weekdays, id: \.self) { weekday in
                                Text(weekday).font(.system(size: 11, weight: .semibold))
                                    .foregroundColor(secondaryInk).frame(maxWidth: .infinity)
                            }
                        }
                        Rectangle().fill(dividerInk).frame(height: 1).accessibilityHidden(true)
                        LazyVGrid(columns: Array(repeating: GridItem(.flexible(), spacing: 4), count: 7), spacing: 3) {
                            ForEach(grid.days, id: \.self) { calendarCell($0, month: grid.monthStart) }
                        }
                    }.padding(14).background(Color.white).clipShape(RoundedRectangle(cornerRadius: 14))
                        .overlay(RoundedRectangle(cornerRadius: 14).strokeBorder(dividerInk))

                    HStack {
                        VStack(alignment: .leading, spacing: 3) {
                            Text(selectedTitle).font(.system(size: 16, weight: .semibold)).lineLimit(1)
                            Text("\(dayTasks.count) 项待办").font(.caption).foregroundColor(secondaryInk)
                        }
                        Spacer()
                        Button { startTask(on: selectedDay) } label: { Label("新建待办", systemImage: "plus") }
                            .buttonStyle(.borderedProminent).controlSize(.regular).accessibilityLabel("在选中日期添加待办")
                    }
                    ForEach(dayTasks) { task in
                        TaskRow(task: task, occurrenceDay: key, planner: planner, edit: { editorTask = task }, errorText: $errorText)
                    }
                }.padding(24)
            }
        }.sheet(item: $newTaskDay) { day in
            TaskEditor(planner: planner, initialDate: day.id)
        }.sheet(item: $editorTask) { TaskEditor(planner: planner, task: $0, initialDate: key) }
        .alert("提示", isPresented: Binding(get: { errorText != nil }, set: { if !$0 { errorText = nil } })) { Button("好", role: .cancel) {} } message: { Text(errorText ?? "") }
    }
    private func calendarCell(_ date: Date, month: Date) -> some View {
        let key = TaskSchedule.ymd(date), selected = Calendar.current.isDate(date, inSameDayAs: selectedDay), inMonth = Calendar.current.component(.month, from: date) == Calendar.current.component(.month, from: month)
        let today = Calendar.current.isDateInToday(date)
        let count = planner.tasks.filter { TaskSchedule.occurs($0, on: key) }.count
        let badges = countdown.selections.compactMap { entry -> String? in
            guard entry.value.start == key, let kind = EventKind(rawValue: entry.key) else { return nil }
            return appearance.name(for: kind)
        }
        return Button { selectedDay = date } label: {
            VStack(spacing: 3) {
                Text("\(Calendar.current.component(.day, from: date))")
                    .font(.system(size: 15, weight: selected || today ? .semibold : .regular))
                    .monospacedDigit()
                    .foregroundColor(selected || today ? macBlue : (inMonth ? ink : secondaryInk))
                HStack(spacing: 3) {
                    if count > 0 { Circle().fill(macBlue).frame(width: 5, height: 5) }
                    if !badges.isEmpty { Circle().fill(.orange).frame(width: 5, height: 5) }
                }.frame(height: 5)
            }
            .frame(maxWidth: .infinity, minHeight: 37)
            .background(RoundedRectangle(cornerRadius: 8).fill(selected ? macBlue.opacity(0.12) : .clear))
            .overlay(RoundedRectangle(cornerRadius: 8).strokeBorder(today ? macBlue.opacity(0.48) : .clear))
            .contentShape(Rectangle())
        }.buttonStyle(.plain)
            .simultaneousGesture(TapGesture(count: 2).onEnded { startTask(on: date) })
            .help("双击添加待办")
            .accessibilityLabel("\(key)\(today ? "，今天" : "")，\(count)项待办\(badges.isEmpty ? "" : "，\(badges.joined(separator: "、"))")")
            .accessibilityAddTraits(selected ? .isSelected : [])
            .accessibilityAction(named: Text("在这一天添加待办")) { startTask(on: date) }
    }
    private func monthButton(_ offset: Int, icon: String, label: String) -> some View {
        Button { changeMonth(offset) } label: {
            Image(systemName: icon).font(.system(size: 12, weight: .semibold))
                .frame(width: 30, height: 28).background(paper, in: RoundedRectangle(cornerRadius: 7))
        }.buttonStyle(.plain).help(label).accessibilityLabel(label)
    }
    private func startTask(on date: Date) {
        guard newTaskDay == nil && editorTask == nil else { return }
        selectedDay = date
        newTaskDay = NewTaskDay(id: TaskSchedule.ymd(date))
    }
    private var monthTitle: String { let f = DateFormatter(); f.locale = Locale(identifier: "zh_CN"); f.dateFormat = "yyyy 年 M 月"; return f.string(from: visibleMonth) }
    private var selectedTitle: String { let f = DateFormatter(); f.locale = Locale(identifier: "zh_CN"); f.dateFormat = "M月d日 · EEE"; return f.string(from: selectedDay) }
    private func changeMonth(_ amount: Int) { if let date = Calendar.current.date(byAdding: .month, value: amount, to: visibleMonth) { visibleMonth = date; selectedDay = Calendar.current.date(from: Calendar.current.dateComponents([.year, .month], from: date)) ?? date } }
}

private struct NotesView: View {
    @ObservedObject var planner: PlannerStore
    @State(wrappedValue: "") private var search: String
    @State(wrappedValue: nil) private var editingNote: PlannerNote?
    @State(wrappedValue: false) private var showingNew: Bool
    @State(wrappedValue: nil) private var errorText: String?
    @State(wrappedValue: nil) private var successText: String?
    private var filtered: [PlannerNote] { (search.isEmpty ? planner.notes : planner.notes.filter { $0.title.localizedCaseInsensitiveContains(search) || $0.body.localizedCaseInsensitiveContains(search) }).sorted { $0.updatedAt > $1.updatedAt } }
    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            PageHeader(title: "笔记", subtitle: "记录想法，也可以生成关联待办") { Button { showingNew = true } label: { Label("新建笔记", systemImage: "square.and.pencil") }.accessibilityLabel("新建笔记") }
            VStack(spacing: 14) {
                TextField("搜索笔记", text: $search).textFieldStyle(.roundedBorder)
                ScrollView {
                    LazyVStack(spacing: 10) {
                        if filtered.isEmpty { Text(search.isEmpty ? "还没有笔记" : "没有匹配的笔记").foregroundColor(secondaryInk).padding(40) }
                        ForEach(filtered) { note in
                            VStack(alignment: .leading, spacing: 8) {
                                Button { editingNote = note } label: { VStack(alignment: .leading, spacing: 5) { Text(note.title.isEmpty ? "无标题" : note.title).font(.headline); Text(note.body.isEmpty ? "空笔记" : note.body).font(.subheadline).foregroundColor(secondaryInk).lineLimit(3) }.frame(maxWidth: .infinity, alignment: .leading).contentShape(Rectangle()) }.buttonStyle(.plain).accessibilityLabel("编辑笔记：\(note.title.isEmpty ? "无标题" : note.title)")
                                HStack { Spacer(); Button("生成待办") { do { _ = try planner.createTask(from: note.id, date: TaskSchedule.ymd(Date())); successText = "已生成今天的待办，可在“今天”和“日历”中查看。" } catch { errorText = error.localizedDescription } }.controlSize(.small).accessibilityLabel("从笔记生成今天的待办") }
                            }.padding(14).background(Color.white).cornerRadius(12)
                        }
                    }
                }
            }.padding(24)
        }.sheet(isPresented: $showingNew) { NoteEditor(planner: planner) }.sheet(item: $editingNote) { NoteEditor(planner: planner, note: $0) }
        .alert(errorText == nil ? "已生成待办" : "提示", isPresented: Binding(get: { errorText != nil || successText != nil }, set: { if !$0 { errorText = nil; successText = nil } })) { Button("好", role: .cancel) {} } message: { Text(errorText ?? successText ?? "") }
    }
}

private struct TaskEditor: View {
    @ObservedObject var planner: PlannerStore
    let existing: PlannerTask?
    @Environment(\.dismiss) private var dismiss
    @State(wrappedValue: "") private var title: String
    @State(wrappedValue: "") private var details: String
    @State(wrappedValue: false) private var hasDate: Bool
    @State(wrappedValue: Date()) private var date: Date
    @State(wrappedValue: false) private var hasReminder: Bool
    @State(wrappedValue: Date()) private var reminder: Date
    @State(wrappedValue: TaskRecurrence.none) private var recurrence: TaskRecurrence
    @State(wrappedValue: false) private var priority: Bool
    @State(wrappedValue: nil) private var errorText: String?
    @State(wrappedValue: false) private var confirmDelete: Bool
    @FocusState private var titleFocused: Bool
    init(planner: PlannerStore, task: PlannerTask? = nil, initialDate: String? = nil) {
        self.planner = planner; existing = task
        let seededDate = TaskEditorDefaults.initialDate(existing: task, suggestedDate: initialDate)
        let dateValue = seededDate.flatMap(CountdownMath.date) ?? Date()
        _title = State(initialValue: task?.title ?? ""); _details = State(initialValue: task?.details ?? ""); _hasDate = State(initialValue: seededDate != nil); _date = State(initialValue: dateValue)
        let timeBits: [Int] = task?.reminderTime?.split(separator: ":").compactMap { Int(String($0)) } ?? []
        let reminderDate = timeBits.count == 2 ? (Calendar.current.date(bySettingHour: timeBits[0], minute: timeBits[1], second: 0, of: Date()) ?? Date()) : (Calendar.current.date(bySettingHour: 9, minute: 0, second: 0, of: Date()) ?? Date())
        _hasReminder = State(initialValue: task?.reminderTime != nil); _reminder = State(initialValue: reminderDate); _recurrence = State(initialValue: task?.recurrence ?? .none); _priority = State(initialValue: task?.isPriority ?? false)
    }
    var body: some View {
        VStack(alignment: .leading, spacing: 14) {
            Text(existing == nil ? "新建待办" : "编辑待办").font(.title2.bold())
            TextField("标题", text: $title).textFieldStyle(.roundedBorder).focused($titleFocused)
            TextField("详细说明（可选）", text: $details, axis: .vertical).lineLimit(3...6).textFieldStyle(.roundedBorder)
            Toggle("安排日期", isOn: $hasDate).onChange(of: hasDate) { if !$0 { hasReminder = false; recurrence = .none } }
            if hasDate { DatePicker("日期", selection: $date, displayedComponents: .date) }
            Toggle("提醒", isOn: $hasReminder).disabled(!hasDate)
            if hasReminder { DatePicker("提醒时间", selection: $reminder, displayedComponents: .hourAndMinute) }
            Picker("重复", selection: $recurrence) { ForEach(TaskRecurrence.allCases) { Text($0 == .weekdays ? "工作日（周一至周五）" : $0.title).tag($0) } }.disabled(!hasDate)
            Toggle("标为重要", isOn: $priority)
            if let errorText { Text(errorText).foregroundColor(.red).font(.caption) }
            Spacer()
            HStack { if existing != nil { Button("删除", role: .destructive) { confirmDelete = true }.accessibilityLabel("删除待办") }; Spacer(); Button("取消") { dismiss() }.keyboardShortcut(.cancelAction); Button("保存", action: save).keyboardShortcut(.defaultAction).disabled(title.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty) }
        }.padding(24).frame(width: 430).frame(minHeight: 470).background(paper)
        .onAppear { if existing == nil { titleFocused = true } }
        .alert("删除这项待办？", isPresented: $confirmDelete) {
            Button("取消", role: .cancel) {}
            Button("删除", role: .destructive) {
                guard let existing else { return }
                do { try planner.deleteTask(id: existing.id); dismiss() }
                catch { errorText = error.localizedDescription }
            }
        } message: {
            Text(existing?.recurrence == TaskRecurrence.none ? "将从待办和日历中移除。" : "该重复待办的所有日期都会一并移除。")
        }
    }
    private func save() {
        let time: String? = hasDate && hasReminder ? { let c = Calendar.current.dateComponents([.hour, .minute], from: reminder); return String(format: "%02d:%02d", c.hour ?? 0, c.minute ?? 0) }() : nil
        var task = existing ?? PlannerTask(title: title); task.title = title.trimmingCharacters(in: .whitespacesAndNewlines); task.details = details; task.startDate = hasDate ? TaskSchedule.ymd(date) : nil; task.reminderTime = time; task.recurrence = hasDate ? recurrence : .none; task.isPriority = priority
        do { existing == nil ? try planner.addTask(task) : try planner.updateTask(task); dismiss() } catch { errorText = error.localizedDescription }
    }
}

private struct NoteEditor: View {
    @ObservedObject var planner: PlannerStore
    let existing: PlannerNote?
    @Environment(\.dismiss) private var dismiss
    @State(wrappedValue: "") private var title: String
    @State(wrappedValue: "") private var bodyText: String
    @State(wrappedValue: "") private var baselineTitle: String
    @State(wrappedValue: "") private var baselineBody: String
    @State(wrappedValue: nil) private var errorText: String?
    @State(wrappedValue: false) private var confirmDiscard: Bool
    @State(wrappedValue: false) private var confirmDelete: Bool
    private var dirty: Bool { title != baselineTitle || bodyText != baselineBody }
    init(planner: PlannerStore, note: PlannerNote? = nil) {
        self.planner = planner; existing = note
        let initialTitle = note?.title ?? "", initialBody = note?.body ?? ""
        _title = State(initialValue: initialTitle); _bodyText = State(initialValue: initialBody)
        _baselineTitle = State(initialValue: initialTitle); _baselineBody = State(initialValue: initialBody)
    }
    var body: some View {
        VStack(alignment: .leading, spacing: 14) {
            Text(existing == nil ? "新建笔记" : "编辑笔记").font(.title2.bold())
            TextField("标题", text: $title).textFieldStyle(.roundedBorder)
            TextEditor(text: $bodyText).font(.body).padding(5).background(Color.white).cornerRadius(8).overlay(RoundedRectangle(cornerRadius: 8).stroke(Color.black.opacity(0.12)))
            if let errorText { Text(errorText).foregroundColor(.red).font(.caption) }
            HStack { if existing != nil { Button("删除", role: .destructive) { confirmDelete = true } }; Spacer(); Button("取消") { if dirty { confirmDiscard = true } else { dismiss() } }.keyboardShortcut(.cancelAction); Button("保存") { _ = save(closeAfter: true) }.keyboardShortcut(.defaultAction) }
        }.padding(24).frame(width: 500, height: 460).background(paper).interactiveDismissDisabled(dirty)
        .alert("放弃未保存的修改？", isPresented: $confirmDiscard) { Button("继续编辑", role: .cancel) {}; Button("放弃修改", role: .destructive) { dismiss() } }
        .alert("删除这篇笔记？", isPresented: $confirmDelete) {
            Button("取消", role: .cancel) {}
            Button("删除", role: .destructive) {
                guard let existing else { return }
                do { try planner.deleteNote(id: existing.id); dismiss() }
                catch { errorText = error.localizedDescription }
            }
        }
        .onAppear {
            UnsavedNoteGuard.shared.register { save(closeAfter: false) }
            UnsavedNoteGuard.shared.setDirty(dirty)
        }
        .onChange(of: title) { _ in UnsavedNoteGuard.shared.setDirty(dirty) }
        .onChange(of: bodyText) { _ in UnsavedNoteGuard.shared.setDirty(dirty) }
        .onDisappear { UnsavedNoteGuard.shared.unregister() }
    }
    private func save(closeAfter: Bool) -> Bool {
        var note = existing ?? PlannerNote(title: title, body: bodyText); note.title = title; note.body = bodyText; note.updatedAt = PlannerClock.timestamp()
        do {
            existing == nil ? try planner.addNote(note) : try planner.updateNote(note)
            baselineTitle = title; baselineBody = bodyText; UnsavedNoteGuard.shared.setDirty(false)
            if closeAfter { dismiss() }
            return true
        } catch { errorText = error.localizedDescription; return false }
    }
}

private struct PageHeader<Actions: View>: View {
    let title: String, subtitle: String
    @ViewBuilder let actions: () -> Actions
    var body: some View {
        HStack(spacing: 12) {
            VStack(alignment: .leading, spacing: 4) {
                Text(title).font(.system(size: 24, weight: .semibold)).foregroundColor(ink)
                Text(subtitle).font(.system(size: 12, weight: .medium)).foregroundColor(secondaryInk).lineLimit(1)
            }
            Spacer(minLength: 8)
            actions()
        }.padding(.horizontal, 24).padding(.top, 18).padding(.bottom, 12)
    }
}

private struct ReminderFooter: View {
    var body: some View { VStack(alignment: .leading, spacing: 7) { Label("关闭窗口后仍在菜单栏运行，退出有盼后停止提醒。", systemImage: "menubar.rectangle").font(.caption).foregroundColor(secondaryInk); Button("测试提醒") { ReminderCoordinator.shared.showTestReminder() }.controlSize(.small).accessibilityLabel("测试提醒，不更改待办状态") }.padding(.top, 4) }
}

private struct SunriseScene: View {
    var body: some View { Canvas { context, size in
        let w = size.width, h = size.height
        context.fill(Path(ellipseIn: CGRect(x: w * 0.59, y: h * 0.08, width: w * 0.25, height: w * 0.25)), with: .color(Color(red: 0.79, green: 0.88, blue: 1).opacity(0.9)))
        var far = Path(); far.move(to: CGPoint(x: 0, y: h * 0.72)); far.addLine(to: CGPoint(x: w * 0.38, y: h * 0.3)); far.addLine(to: CGPoint(x: w * 0.73, y: h * 0.7)); far.addLine(to: CGPoint(x: w, y: h * 0.48)); far.addLine(to: CGPoint(x: w, y: h)); far.addLine(to: CGPoint(x: 0, y: h)); far.closeSubpath(); context.fill(far, with: .color(Color(red: 0.58, green: 0.69, blue: 0.94)))
        var near = Path(); near.move(to: CGPoint(x: 0, y: h * 0.88)); near.addLine(to: CGPoint(x: w * 0.36, y: h * 0.58)); near.addLine(to: CGPoint(x: w * 0.62, y: h * 0.83)); near.addLine(to: CGPoint(x: w, y: h * 0.55)); near.addLine(to: CGPoint(x: w, y: h)); near.addLine(to: CGPoint(x: 0, y: h)); near.closeSubpath(); context.fill(near, with: .color(Color(red: 0.35, green: 0.51, blue: 0.85)))
    }.clipShape(RoundedRectangle(cornerRadius: 9)) }
}

private struct AppearanceSettingsView: View {
    @ObservedObject var appearance: AppearanceStore
    @Environment(\.dismiss) private var dismiss
    @State private var names: [String: String]
    @State(wrappedValue: nil) private var errorText: String?
    @State(wrappedValue: nil) private var savedStatus: String?
    @State(wrappedValue: false) private var confirmClose: Bool

    private var namesChanged: Bool {
        EventKind.allCases.contains { (names[$0.rawValue] ?? $0.title) != appearance.name(for: $0) }
    }

    init(appearance: AppearanceStore) {
        self.appearance = appearance
        _names = State(initialValue: Dictionary(uniqueKeysWithValues: EventKind.allCases.map { ($0.rawValue, appearance.name(for: $0)) }))
    }

    var body: some View {
        let viewport = sheetViewport
        let contentWidth = max(540, viewport.width)
        return ScrollView([.horizontal, .vertical]) {
            VStack(alignment: .leading, spacing: 18) {
            HStack {
                VStack(alignment: .leading, spacing: 3) {
                    Text("外观与倒计时名称").font(.title2.bold())
                    Text("窗口大小和透明度会立即保存并应用。").font(.caption).foregroundColor(secondaryInk)
                }
                Spacer()
                Button("完成") { if namesChanged { confirmClose = true } else { dismiss() } }.keyboardShortcut(.cancelAction)
            }

            GroupBox("窗口大小") {
                VStack(alignment: .leading, spacing: 10) {
                    Text("直接拖动窗口边角调整大小，松开后自动保存。")
                    Text("已保存尺寸：\(Int(appearance.width.rounded())) × \(Int(appearance.height.rounded())) 点")
                        .font(.caption).foregroundColor(secondaryInk)
                }.frame(maxWidth: .infinity, alignment: .leading).padding(.vertical, 4)
            }

            GroupBox("窗口透明度") {
                HStack(spacing: 12) {
                    Slider(value: Binding(get: { appearance.opacity }, set: updateOpacity), in: 0.5...1.0, step: 0.01)
                        .accessibilityLabel("窗口透明度")
                    Text("\(Int((appearance.opacity * 100).rounded()))%")
                        .monospacedDigit().frame(width: 44, alignment: .trailing)
                }.padding(.vertical, 4)
            }

            GroupBox("五个倒计时名称") {
                VStack(alignment: .leading, spacing: 9) {
                    ForEach(Array(EventKind.allCases.enumerated()), id: \.element.id) { index, kind in
                        HStack {
                            Text(index == 0 ? "主倒计时" : "倒计时 \(index + 1)").frame(width: 74, alignment: .leading).foregroundColor(secondaryInk)
                            TextField("1–24 个字符", text: nameBinding(for: kind)).textFieldStyle(.roundedBorder)
                                .accessibilityLabel("\(index == 0 ? "主倒计时" : "倒计时 \(index + 1)")名称")
                        }
                    }
                    HStack {
                        Button("恢复默认名称") { resetNames() }
                        Spacer()
                        Button("保存名称") { _ = saveNames() }.keyboardShortcut(.defaultAction)
                            .disabled(!namesChanged)
                    }.padding(.top, 3)
                }.padding(.vertical, 4)
            }

            if let errorText { Text(errorText).foregroundColor(.red).font(.caption) }
            if let savedStatus { Label(savedStatus, systemImage: "checkmark.circle").foregroundColor(.green).font(.caption) }
            HStack {
                Button("恢复默认外观") { resetAppearance() }
                    .help("恢复标准窗口大小和 100% 透明度；不会修改日期、任务或名称")
                Spacer()
            }
            }.padding(24).frame(width: contentWidth, alignment: .leading)
        }.frame(width: viewport.width, height: viewport.height).background(paper)
        .interactiveDismissDisabled(namesChanged)
        .alert("保存修改后的名称？", isPresented: $confirmClose) {
            Button("继续编辑", role: .cancel) {}
            Button("放弃修改", role: .destructive) { dismiss() }
            Button("保存并关闭") { if saveNames() { dismiss() } }
        }
    }

    private var sheetViewport: CGSize {
        let visible = AppDelegate.current?.mainWindow?.screen?.visibleFrame.size ?? NSScreen.main?.visibleFrame.size ?? CGSize(width: 540, height: 590)
        return CGSize(width: min(540, max(280, visible.width - 32)), height: min(590, max(320, visible.height - 32)))
    }

    private func nameBinding(for kind: EventKind) -> Binding<String> {
        Binding(get: { names[kind.rawValue] ?? kind.title }, set: { names[kind.rawValue] = $0; savedStatus = nil })
    }

    private func updateOpacity(_ value: Double) {
        do { try appearance.updateOpacity(value); errorText = nil }
        catch { errorText = "透明度保存失败：\(error.localizedDescription)" }
    }

    private func saveNames() -> Bool {
        do {
            try appearance.updateNames(names)
            names = Dictionary(uniqueKeysWithValues: EventKind.allCases.map { ($0.rawValue, appearance.name(for: $0)) })
            errorText = nil
            savedStatus = "名称已保存"
            return true
        } catch {
            savedStatus = nil
            errorText = "名称保存失败：\(error.localizedDescription)"
            return false
        }
    }

    private func resetNames() {
        do {
            try appearance.resetNames()
            names = Dictionary(uniqueKeysWithValues: EventKind.allCases.map { ($0.rawValue, appearance.name(for: $0)) })
            errorText = nil
            savedStatus = "已恢复默认名称"
        } catch { errorText = "默认名称恢复失败：\(error.localizedDescription)" }
    }

    private func resetAppearance() {
        do { try appearance.resetAppearance(); errorText = nil }
        catch { errorText = "默认外观恢复失败：\(error.localizedDescription)" }
    }
}

private struct EventEditor: View {
    let kind: EventKind
    let name: String
    @ObservedObject var store: CountdownStore
    @Environment(\.dismiss) private var dismiss
    @State(wrappedValue: Date()) private var start: Date
    @State(wrappedValue: Date()) private var end: Date
    @State(wrappedValue: false) private var hasEnd: Bool
    @State(wrappedValue: nil) private var errorText: String?
    @State(wrappedValue: false) private var confirmReset: Bool
    init(kind: EventKind, name: String, store: CountdownStore, today: Date) { self.kind = kind; self.name = name; self.store = store; let saved = store.selections[kind.rawValue], initial = saved.flatMap { CountdownMath.date($0.start) } ?? CountdownMath.defaultStart(kind, today: today) ?? today; _start = State(initialValue: initial); _end = State(initialValue: saved?.end.flatMap(CountdownMath.date) ?? initial); _hasEnd = State(initialValue: saved?.end != nil) }
    var body: some View {
        VStack(alignment: .leading, spacing: 17) {
            Text("设置\(name)日期").font(.title2.bold()); DatePicker(kind == .retirement ? "目标日期" : "开始日期", selection: $start, displayedComponents: .date)
            if kind != .retirement { Toggle("设置结束日期", isOn: $hasEnd); if hasEnd { DatePicker("结束日期", selection: $end, in: start..., displayedComponents: .date) } }
            if let errorText { Text(errorText).foregroundColor(.red).font(.caption) }
            Spacer(); HStack { Button(kind == .labor || kind == .national ? "恢复默认" : "清空日期") { confirmReset = true }; Spacer(); Button("取消") { dismiss() }; Button("保存") { update(EventDates(start: CountdownMath.ymd(start), end: kind != .retirement && hasEnd ? CountdownMath.ymd(end) : nil)) }.keyboardShortcut(.defaultAction) }
        }.padding(24).frame(width: 400).frame(minHeight: 255).background(paper)
        .alert(kind == .labor || kind == .national ? "恢复默认日期？" : "清空\(name)日期？", isPresented: $confirmReset) {
            Button("取消", role: .cancel) {}
            Button("确认", role: .destructive) { update(nil) }
        }
    }
    private func update(_ dates: EventDates?) { do { try store.update(kind, dates: dates); dismiss() } catch { errorText = error.localizedDescription } }
}
