import fs from 'node:fs/promises'
import path from 'node:path'

const LOCALES_DIR = path.resolve('src/i18n/locales')

function stableStringify(obj) {
  return `${JSON.stringify(obj, null, 2)}\n`
}

const key =
  'Please complete the bank transfer first, then upload a clear transfer receipt. The balance will be credited after manual review.'

const commonTranslations = {
  en: {
    Rejected: 'Rejected',
    'Reject Order': 'Reject Order',
    'Rejection reason': 'Rejection reason',
    'Enter rejection reason': 'Enter rejection reason',
    'Please provide a rejection reason': 'Please provide a rejection reason',
    'Order rejected successfully': 'Order rejected successfully',
    'Failed to reject order': 'Failed to reject order',
    'Confirm rejection': 'Confirm rejection',
    'The rejection reason will be visible to the user. This action does not credit their balance.':
      'The rejection reason will be visible to the user. This action does not credit their balance.',
    'Corporate contact phone': 'Corporate contact phone',
    'Corporate contact WeChat': 'Corporate contact WeChat',
    'Visible user groups': 'Visible user groups',
    'Leave all groups unselected to show corporate payment to every user group.':
      'Leave all groups unselected to show corporate payment to every user group.',
    'No user groups configured': 'No user groups configured',
    'Contact WeChat': 'Contact WeChat',
    'e.g., +86 138 0000 0000': 'e.g., +86 138 0000 0000',
    'e.g., finance_support': 'e.g., finance_support',
    Canceled: 'Canceled',
    'Withdraw Order': 'Withdraw Order',
    'Order withdrawn successfully': 'Order withdrawn successfully',
    'Failed to withdraw order': 'Failed to withdraw order',
    'Confirm withdrawal': 'Confirm withdrawal',
    'Withdraw this pending corporate payment order? The uploaded proof will remain available in the order history.':
      'Withdraw this pending corporate payment order? The uploaded proof will remain available in the order history.',
  },
  zh: {
    Rejected: '已拒绝',
    'Reject Order': '拒绝订单',
    'Rejection reason': '拒绝原因',
    'Enter rejection reason': '请输入拒绝原因',
    'Please provide a rejection reason': '请填写拒绝原因',
    'Order rejected successfully': '订单已拒绝',
    'Failed to reject order': '拒绝订单失败',
    'Confirm rejection': '确认拒绝',
    'The rejection reason will be visible to the user. This action does not credit their balance.':
      '拒绝原因将对用户可见，此操作不会增加用户余额。',
    'Corporate contact phone': '对公联系电话',
    'Corporate contact WeChat': '对公联系微信',
    'Visible user groups': '可见用户分组',
    'Leave all groups unselected to show corporate payment to every user group.':
      '不选择任何分组时，对公支付将对所有用户分组显示。',
    'No user groups configured': '尚未配置用户分组',
    'Contact WeChat': '联系微信',
    'e.g., +86 138 0000 0000': '例如：+86 138 0000 0000',
    'e.g., finance_support': '例如：finance_support',
    Canceled: '已撤回',
    'Withdraw Order': '撤回订单',
    'Order withdrawn successfully': '订单已撤回',
    'Failed to withdraw order': '撤回订单失败',
    'Confirm withdrawal': '确认撤回',
    'Withdraw this pending corporate payment order? The uploaded proof will remain available in the order history.':
      '确定撤回这个待确认的对公支付订单吗？已上传的凭证仍会保留在订单记录中。',
  },
  'zh-TW': {
    Rejected: '已拒絕',
    'Reject Order': '拒絕訂單',
    'Rejection reason': '拒絕原因',
    'Enter rejection reason': '請輸入拒絕原因',
    'Please provide a rejection reason': '請填寫拒絕原因',
    'Order rejected successfully': '訂單已拒絕',
    'Failed to reject order': '拒絕訂單失敗',
    'Confirm rejection': '確認拒絕',
    'The rejection reason will be visible to the user. This action does not credit their balance.':
      '拒絕原因將對使用者可見，此操作不會增加使用者餘額。',
    'Corporate contact phone': '對公聯絡電話',
    'Corporate contact WeChat': '對公聯絡微信',
    'Visible user groups': '可見使用者群組',
    'Leave all groups unselected to show corporate payment to every user group.':
      '不選擇任何群組時，對公支付將對所有使用者群組顯示。',
    'No user groups configured': '尚未設定使用者群組',
    'Contact WeChat': '聯絡微信',
    'e.g., +86 138 0000 0000': '例如：+86 138 0000 0000',
    'e.g., finance_support': '例如：finance_support',
    Canceled: '已撤回',
    'Withdraw Order': '撤回訂單',
    'Order withdrawn successfully': '訂單已撤回',
    'Failed to withdraw order': '撤回訂單失敗',
    'Confirm withdrawal': '確認撤回',
    'Withdraw this pending corporate payment order? The uploaded proof will remain available in the order history.':
      '確定撤回這個待確認的對公支付訂單嗎？已上傳的憑證仍會保留在訂單記錄中。',
  },
  fr: {
    Rejected: 'Refusé',
    'Reject Order': 'Refuser la commande',
    'Rejection reason': 'Motif du refus',
    'Enter rejection reason': 'Saisissez le motif du refus',
    'Please provide a rejection reason': 'Veuillez indiquer un motif de refus',
    'Order rejected successfully': 'Commande refusée',
    'Failed to reject order': 'Échec du refus de la commande',
    'Confirm rejection': 'Confirmer le refus',
    'The rejection reason will be visible to the user. This action does not credit their balance.':
      'Le motif du refus sera visible par l’utilisateur. Cette action ne crédite pas son solde.',
    'Corporate contact phone': 'Téléphone du contact entreprise',
    'Corporate contact WeChat': 'WeChat du contact entreprise',
    'Visible user groups': 'Groupes d’utilisateurs visibles',
    'Leave all groups unselected to show corporate payment to every user group.':
      'Ne sélectionnez aucun groupe pour afficher le paiement entreprise à tous les groupes.',
    'No user groups configured': 'Aucun groupe d’utilisateurs configuré',
    'Contact WeChat': 'WeChat du contact',
    'e.g., +86 138 0000 0000': 'p. ex. +86 138 0000 0000',
    'e.g., finance_support': 'p. ex. finance_support',
    Canceled: 'Retiré',
    'Withdraw Order': 'Retirer la commande',
    'Order withdrawn successfully': 'Commande retirée',
    'Failed to withdraw order': 'Échec du retrait de la commande',
    'Confirm withdrawal': 'Confirmer le retrait',
    'Withdraw this pending corporate payment order? The uploaded proof will remain available in the order history.':
      'Retirer cette commande de paiement entreprise en attente ? Le justificatif restera disponible dans l’historique.',
  },
  ja: {
    Rejected: '却下済み',
    'Reject Order': '注文を却下',
    'Rejection reason': '却下理由',
    'Enter rejection reason': '却下理由を入力してください',
    'Please provide a rejection reason': '却下理由を入力してください',
    'Order rejected successfully': '注文を却下しました',
    'Failed to reject order': '注文の却下に失敗しました',
    'Confirm rejection': '却下を確定',
    'The rejection reason will be visible to the user. This action does not credit their balance.':
      '却下理由はユーザーに表示されます。この操作では残高は追加されません。',
    'Corporate contact phone': '法人支払い連絡先電話番号',
    'Corporate contact WeChat': '法人支払い連絡先WeChat',
    'Visible user groups': '表示するユーザーグループ',
    'Leave all groups unselected to show corporate payment to every user group.':
      'グループを選択しない場合、法人支払いはすべてのユーザーグループに表示されます。',
    'No user groups configured': 'ユーザーグループが設定されていません',
    'Contact WeChat': '連絡先WeChat',
    'e.g., +86 138 0000 0000': '例：+86 138 0000 0000',
    'e.g., finance_support': '例：finance_support',
    Canceled: '撤回済み',
    'Withdraw Order': '注文を撤回',
    'Order withdrawn successfully': '注文を撤回しました',
    'Failed to withdraw order': '注文の撤回に失敗しました',
    'Confirm withdrawal': '撤回を確定',
    'Withdraw this pending corporate payment order? The uploaded proof will remain available in the order history.':
      'この確認待ちの法人支払い注文を撤回しますか？アップロード済みの証憑は注文履歴に残ります。',
  },
  ru: {
    Rejected: 'Отклонено',
    'Reject Order': 'Отклонить заказ',
    'Rejection reason': 'Причина отклонения',
    'Enter rejection reason': 'Укажите причину отклонения',
    'Please provide a rejection reason': 'Укажите причину отклонения',
    'Order rejected successfully': 'Заказ отклонён',
    'Failed to reject order': 'Не удалось отклонить заказ',
    'Confirm rejection': 'Подтвердить отклонение',
    'The rejection reason will be visible to the user. This action does not credit their balance.':
      'Причина отклонения будет видна пользователю. Это действие не пополняет баланс.',
    'Corporate contact phone': 'Телефон для корпоративных платежей',
    'Corporate contact WeChat': 'WeChat для корпоративных платежей',
    'Visible user groups': 'Доступные группы пользователей',
    'Leave all groups unselected to show corporate payment to every user group.':
      'Не выбирайте группы, чтобы показать корпоративную оплату всем пользователям.',
    'No user groups configured': 'Группы пользователей не настроены',
    'Contact WeChat': 'WeChat для связи',
    'e.g., +86 138 0000 0000': 'например, +86 138 0000 0000',
    'e.g., finance_support': 'например, finance_support',
    Canceled: 'Отозвано',
    'Withdraw Order': 'Отозвать заказ',
    'Order withdrawn successfully': 'Заказ отозван',
    'Failed to withdraw order': 'Не удалось отозвать заказ',
    'Confirm withdrawal': 'Подтвердить отзыв',
    'Withdraw this pending corporate payment order? The uploaded proof will remain available in the order history.':
      'Отозвать этот ожидающий корпоративный платёж? Загруженный документ останется в истории заказов.',
  },
  vi: {
    Rejected: 'Đã từ chối',
    'Reject Order': 'Từ chối đơn hàng',
    'Rejection reason': 'Lý do từ chối',
    'Enter rejection reason': 'Nhập lý do từ chối',
    'Please provide a rejection reason': 'Vui lòng nhập lý do từ chối',
    'Order rejected successfully': 'Đã từ chối đơn hàng',
    'Failed to reject order': 'Không thể từ chối đơn hàng',
    'Confirm rejection': 'Xác nhận từ chối',
    'The rejection reason will be visible to the user. This action does not credit their balance.':
      'Người dùng sẽ thấy lý do từ chối. Thao tác này không cộng số dư.',
    'Corporate contact phone': 'Số điện thoại liên hệ doanh nghiệp',
    'Corporate contact WeChat': 'WeChat liên hệ doanh nghiệp',
    'Visible user groups': 'Nhóm người dùng được hiển thị',
    'Leave all groups unselected to show corporate payment to every user group.':
      'Không chọn nhóm nào để hiển thị thanh toán doanh nghiệp cho mọi nhóm người dùng.',
    'No user groups configured': 'Chưa cấu hình nhóm người dùng',
    'Contact WeChat': 'WeChat liên hệ',
    'e.g., +86 138 0000 0000': 'ví dụ: +86 138 0000 0000',
    'e.g., finance_support': 'ví dụ: finance_support',
    Canceled: 'Đã rút lại',
    'Withdraw Order': 'Rút lại đơn hàng',
    'Order withdrawn successfully': 'Đã rút lại đơn hàng',
    'Failed to withdraw order': 'Không thể rút lại đơn hàng',
    'Confirm withdrawal': 'Xác nhận rút lại',
    'Withdraw this pending corporate payment order? The uploaded proof will remain available in the order history.':
      'Rút lại đơn thanh toán doanh nghiệp đang chờ này? Chứng từ đã tải lên vẫn được lưu trong lịch sử đơn hàng.',
  },
}

const newKeys = {
  en: {
    [key]: key,
    ...commonTranslations.en,
  },
  zh: {
    [key]:
      '请先完成银行转账，然后上传清晰的转账凭证。人工审核通过后，余额将到账。',
    ...commonTranslations.zh,
  },
  'zh-TW': {
    [key]:
      '請先完成銀行轉帳，然後上傳清晰的轉帳憑證。人工審核通過後，餘額將入帳。',
    ...commonTranslations['zh-TW'],
  },
  fr: {
    [key]:
      'Effectuez d’abord le virement bancaire, puis téléversez un justificatif lisible. Le solde sera crédité après vérification manuelle.',
    ...commonTranslations.fr,
  },
  ja: {
    [key]:
      '先に銀行振込を完了し、鮮明な振込証憑をアップロードしてください。手動審査後に残高へ反映されます。',
    ...commonTranslations.ja,
  },
  ru: {
    [key]:
      'Сначала выполните банковский перевод, затем загрузите четкий документ, подтверждающий перевод. Баланс будет пополнен после ручной проверки.',
    ...commonTranslations.ru,
  },
  vi: {
    [key]:
      'Vui lòng hoàn tất chuyển khoản ngân hàng trước, sau đó tải lên chứng từ chuyển khoản rõ ràng. Số dư sẽ được cộng sau khi kiểm duyệt thủ công.',
    ...commonTranslations.vi,
  },
}

async function main() {
  let totalApplied = 0

  for (const [locale, translations] of Object.entries(newKeys)) {
    const filePath = path.join(LOCALES_DIR, `${locale}.json`)
    const json = JSON.parse(await fs.readFile(filePath, 'utf8'))
    let count = 0

    for (const [translationKey, value] of Object.entries(translations)) {
      if (json.translation[translationKey] !== value) {
        json.translation[translationKey] = value
        count++
      }
    }

    if (count > 0) {
      json.translation = Object.fromEntries(
        Object.entries(json.translation).sort(([a], [b]) => a.localeCompare(b))
      )
      await fs.writeFile(filePath, stableStringify(json), 'utf8')
    }

    console.log(`${locale}: ${count} translations applied`)
    totalApplied += count
  }

  console.log(`\nTotal: ${totalApplied} translations applied`)
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
