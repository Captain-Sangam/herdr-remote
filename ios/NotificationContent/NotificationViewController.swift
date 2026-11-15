import UIKit
import UserNotifications
import UserNotificationsUI

/// The content extension (decisions 11 and 12 of the plan). For now it only reads the store
/// item from the shared keychain group: the step 6 check of the shared access group on a real
/// device. Step 11 of the plan adds the prompt and the buttons.
final class NotificationViewController: UIViewController, UNNotificationContentExtension {
  private let label = UILabel()

  override func viewDidLoad() {
    super.viewDidLoad()
    label.numberOfLines = 0
    label.font = UIFont(name: "Menlo", size: 13)
    label.translatesAutoresizingMaskIntoConstraints = false
    view.addSubview(label)
    NSLayoutConstraint.activate([
      label.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 16),
      label.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -16),
      label.topAnchor.constraint(equalTo: view.topAnchor, constant: 12),
      label.bottomAnchor.constraint(equalTo: view.bottomAnchor, constant: -12),
    ])
  }

  func didReceive(_ notification: UNNotification) {
    // Before the first unlock the item cannot be read: the alert keeps its title only.
    label.text =
      Store.read().map {
        "Store item read. Paired: \($0.paired ? "yes" : "no"). Mac: \($0.macName)."
      }
      ?? "Store item not readable."
  }
}
