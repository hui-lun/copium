import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../features/backtest/backtest_page.dart';
import '../features/market/market_page.dart';
import '../features/portfolio/portfolio_page.dart';
import '../features/screener/screener_page.dart';
import '../features/stock/stock_detail_page.dart';
import '../features/watchlist/watchlist_page.dart';
import 'theme.dart';

final appRouter = GoRouter(
  initialLocation: '/screener',
  routes: [
    StatefulShellRoute.indexedStack(
      builder: (context, state, shell) => HomeShell(shell: shell),
      branches: [
        _branch('/screener', const ScreenerPage()),
        _branch('/backtest', const BacktestPage()),
        _branch('/market', const MarketPage()),
        _branch('/watchlist', const WatchlistPage()),
        _branch('/portfolio', const PortfolioPage()),
      ],
    ),
    // Optional ?buy=YYYY-MM-DD&sell=YYYY-MM-DD marks a backtest trade on the chart.
    GoRoute(
      path: '/stock/:code',
      builder: (context, state) => StockDetailPage(
        code: state.pathParameters['code']!,
        buyDate: DateTime.tryParse(state.uri.queryParameters['buy'] ?? ''),
        sellDate: DateTime.tryParse(state.uri.queryParameters['sell'] ?? ''),
      ),
    ),
  ],
);

StatefulShellBranch _branch(String path, Widget page) =>
    StatefulShellBranch(routes: [GoRoute(path: path, builder: (context, state) => page)]);

/// Bottom navigation shell, styled after the design's tab bar.
class HomeShell extends StatelessWidget {
  const HomeShell({super.key, required this.shell});

  final StatefulNavigationShell shell;

  static const _tabs = [
    (Icons.travel_explore, '選股'),
    (Icons.query_stats, '回測'),
    (Icons.trending_up, '行情'),
    (Icons.star_border, '自選'),
    (Icons.work_outline, '持倉'),
  ];

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: shell,
      bottomNavigationBar: DecoratedBox(
        decoration: const BoxDecoration(
          color: AppColors.surface,
          border: Border(top: BorderSide(color: AppColors.border)),
        ),
        child: SafeArea(
          top: false,
          child: SizedBox(
            height: 64,
            child: Row(
              children: [
                for (var i = 0; i < _tabs.length; i++)
                  Expanded(
                    child: _NavItem(
                      icon: _tabs[i].$1,
                      label: _tabs[i].$2,
                      active: i == shell.currentIndex,
                      onTap: () => shell.goBranch(i, initialLocation: i == shell.currentIndex),
                    ),
                  ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _NavItem extends StatelessWidget {
  const _NavItem({required this.icon, required this.label, required this.active, required this.onTap});

  final IconData icon;
  final String label;
  final bool active;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final color = active ? AppColors.accent : AppColors.muted;
    return Semantics(
      selected: active,
      button: true,
      child: InkWell(
        onTap: onTap,
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(icon, size: 22, color: color),
            const SizedBox(height: 4),
            Text(label, style: TextStyle(fontSize: 12, color: color)),
          ],
        ),
      ),
    );
  }
}
