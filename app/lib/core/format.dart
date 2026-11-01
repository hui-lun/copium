import 'package:flutter/material.dart';
import 'package:intl/intl.dart';

import 'theme.dart';

final _price = NumberFormat('#,##0.00');
final _int = NumberFormat('#,##0');
final _date = DateFormat('yyyy-MM-dd');

String fmtPrice(num v) => v == 0 ? '--' : _price.format(v);

String fmtInt(num v) => _int.format(v.round());

/// "+1,234" / "-1,234".
String fmtSigned(num v) => '${v > 0 ? '+' : ''}${_int.format(v.round())}';

/// "▲ +1.23%" / "▼ -1.23%".
String fmtPct(num pct, {bool withArrow = true}) =>
    '${withArrow ? arrow(pct) : ''}${pct > 0 ? '+' : ''}${pct.toStringAsFixed(2)}%';

/// Shares to board lots: 32,415,000 -> "32,415 張".
String fmtLots(num shares) => '${_int.format(shares ~/ 1000)} 張';

/// NT$ to hundred-millions: "12.34 億".
String fmtYi(num value) => '${(value / 1e8).toStringAsFixed(2)} 億';

String fmtDate(DateTime d) => _date.format(d);

/// Red for up, green for down (Taiwan convention).
Color changeColor(num v) => v > 0 ? AppColors.up : (v < 0 ? AppColors.down : AppColors.muted);

String arrow(num v) => v > 0 ? '▲ ' : (v < 0 ? '▼ ' : '');
