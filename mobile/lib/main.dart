import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import 'app/app_state.dart';
import 'app/call_state.dart';
import 'pages/call_page.dart';
import 'pages/home_page.dart';
import 'pages/login_page.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  final app = AppState();
  await app.init();
  final call = CallState(app);
  try {
    await call.initRenderers();
  } catch (_) {}
  runApp(MultiProvider(
    providers: [
      ChangeNotifierProvider<AppState>.value(value: app),
      ChangeNotifierProvider<CallState>.value(value: call),
    ],
    child: const NexImApp(),
  ));
}

class NexImApp extends StatelessWidget {
  const NexImApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'NexIM',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        useMaterial3: true,
        colorScheme: ColorScheme.fromSeed(seedColor: const Color(0xFF4F6EF7)),
        scaffoldBackgroundColor: Colors.white,
        appBarTheme: const AppBarTheme(
          backgroundColor: Colors.white,
          surfaceTintColor: Colors.white,
          elevation: 0.5,
          centerTitle: false,
        ),
      ),
      home: Consumer<AppState>(
        builder: (context, app, _) => app.isLogin ? const HomePage() : const LoginPage(),
      ),
      builder: (context, child) => Stack(
        children: [
          if (child != null) child,
          const CallOverlay(),
        ],
      ),
    );
  }
}
