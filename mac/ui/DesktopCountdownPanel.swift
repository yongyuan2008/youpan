import SwiftUI
import AppKit
import Combine

private final class DesktopPanel: NSPanel {
    override var canBecomeKey: Bool { false }
    override var canBecomeMain: Bool { false }
}

final class DesktopCountdownPanelController {
    private let frameAutosaveName = "YouPanDesktopCountdownPanel"
    private var panel: DesktopPanel?
    private var hostingView: NSHostingView<DesktopCountdownPanelView>?
    private var screenObserver: NSObjectProtocol?

    init() {
        screenObserver = NotificationCenter.default.addObserver(
            forName: NSApplication.didChangeScreenParametersNotification,
            object: nil,
            queue: .main
        ) { [weak self] _ in self?.capToVisibleScreen() }
    }

    deinit {
        if let screenObserver { NotificationCenter.default.removeObserver(screenObserver) }
    }

    func sync(store: CountdownStore, appearance: AppearanceStore) {
        guard appearance.pinnedToDesktop else {
            panel?.close()
            panel = nil
            hostingView = nil
            return
        }

        let panel = self.panel ?? makePanel()
        if hostingView == nil {
            let hostingView = NSHostingView(rootView: DesktopCountdownPanelView(store: store, appearance: appearance))
            hostingView.frame = panel.contentView?.bounds ?? .zero
            hostingView.autoresizingMask = [.width, .height]
            panel.contentView = hostingView
            self.hostingView = hostingView
        }
        panel.alphaValue = appearance.opacity
        capToVisibleScreen()
        panel.orderFrontRegardless()
    }

    func destroy() {
        panel?.close()
        panel = nil
        hostingView = nil
        if let screenObserver {
            NotificationCenter.default.removeObserver(screenObserver)
            self.screenObserver = nil
        }
    }

    private func makePanel() -> DesktopPanel {
        let size = DesktopPanelGeometry.defaultSize
        let visible = NSScreen.main?.visibleFrame ?? CGRect(x: 0, y: 0, width: 1024, height: 768)
        let origin = CGPoint(x: visible.maxX - size.width - 24, y: visible.maxY - size.height - 42)
        let panel = DesktopPanel(
            contentRect: CGRect(origin: origin, size: size),
            styleMask: [.borderless, .nonactivatingPanel],
            backing: .buffered,
            defer: false
        )
        panel.level = NSWindow.Level(rawValue: Int(CGWindowLevelForKey(.desktopIconWindow)))
        panel.collectionBehavior = [.canJoinAllSpaces, .stationary, .ignoresCycle]
        panel.backgroundColor = .clear
        panel.isOpaque = false
        panel.hasShadow = true
        panel.hidesOnDeactivate = false
        panel.isMovableByWindowBackground = true
        panel.isReleasedWhenClosed = false
        panel.ignoresMouseEvents = false
        _ = panel.setFrameUsingName(frameAutosaveName)
        var restoredFrame = panel.frame
        restoredFrame.size = size
        panel.setFrame(restoredFrame, display: false)
        panel.setFrameAutosaveName(frameAutosaveName)
        self.panel = panel
        return panel
    }

    private func capToVisibleScreen() {
        guard let panel else { return }
        let fallback = NSScreen.main?.visibleFrame ?? CGRect(origin: .zero, size: DesktopPanelGeometry.defaultSize)
        let capped = DesktopPanelGeometry.cappedFrame(panel.frame, visibleFrames: NSScreen.screens.map(\.visibleFrame), fallback: fallback)
        if panel.frame != capped { panel.setFrame(capped, display: true) }
    }
}

struct DesktopCountdownPanelView: View {
    @ObservedObject var store: CountdownStore
    @ObservedObject var appearance: AppearanceStore
    @State private var now = Date()
    private let timer = Timer.publish(every: 60, on: .main, in: .common).autoconnect()

    private func display(_ kind: EventKind) -> CountdownDisplay {
        CountdownMath.display(
            kind,
            custom: store.selections[kind.rawValue],
            today: now,
            customName: appearance.customName(for: kind)
        )
    }

    var body: some View {
        let item = display(.retirement)
        VStack(alignment: .leading, spacing: 9) {
            HStack(spacing: 8) {
                Image(systemName: "hourglass")
                Text(appearance.name(for: .retirement)).font(.system(size: 14, weight: .semibold))
                Spacer(minLength: 4)
                Text(item.caption).font(.system(size: 11, weight: .medium)).opacity(0.78)
            }
            HStack(alignment: .firstTextBaseline, spacing: 5) {
                Text(item.value).font(.system(size: item.value.count > 5 ? 30 : 42, weight: .bold, design: .rounded)).lineLimit(1)
                if !item.unit.isEmpty { Text(item.unit).font(.system(size: 14, weight: .medium)) }
            }
            if !item.dateLine.isEmpty {
                Text(item.dateLine).font(.system(size: 11, weight: .medium)).opacity(0.76).lineLimit(1)
            }
            Rectangle().fill(Color.white.opacity(0.18)).frame(height: 1)
            VStack(spacing: 0) {
                ForEach(Array(EventKind.allCases.dropFirst())) { kind in
                    compactRow(kind)
                    if kind != .national { Rectangle().fill(Color.white.opacity(0.11)).frame(height: 1) }
                }
            }
        }
        .foregroundColor(.white)
        .padding(.horizontal, 18)
        .padding(.vertical, 15)
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .leading)
        .background(
            LinearGradient(
                colors: [Color(red: 0.11, green: 0.24, blue: 0.56), Color(red: 0.20, green: 0.43, blue: 0.78)],
                startPoint: .topLeading,
                endPoint: .bottomTrailing
            )
        )
        .clipShape(RoundedRectangle(cornerRadius: 18, style: .continuous))
        .overlay(RoundedRectangle(cornerRadius: 18, style: .continuous).stroke(Color.white.opacity(0.16)))
        .padding(4)
        .allowsHitTesting(false)
        .onReceive(timer) { now = $0 }
        .onReceive(NotificationCenter.default.publisher(for: Notification.Name.NSCalendarDayChanged)) { _ in now = Date() }
        .onReceive(NotificationCenter.default.publisher(for: Notification.Name.NSSystemTimeZoneDidChange)) { _ in now = Date() }
        .onReceive(NotificationCenter.default.publisher(for: NSApplication.didBecomeActiveNotification)) { _ in now = Date() }
        .onReceive(NSWorkspace.shared.notificationCenter.publisher(for: NSWorkspace.didWakeNotification)) { _ in now = Date() }
    }

    private func compactRow(_ kind: EventKind) -> some View {
        let item = display(kind)
        return HStack(spacing: 8) {
            Text(appearance.name(for: kind))
                .font(.system(size: 12, weight: .semibold))
                .lineLimit(1)
            Spacer(minLength: 6)
            VStack(alignment: .trailing, spacing: 1) {
                Text(item.unit.isEmpty ? item.value : "\(item.value)\(item.unit)")
                    .font(.system(size: 13, weight: .bold, design: .rounded))
                    .lineLimit(1)
                Text(item.caption)
                    .font(.system(size: 9, weight: .medium))
                    .opacity(0.68)
                    .lineLimit(1)
            }
        }
        .frame(height: 37)
    }
}
