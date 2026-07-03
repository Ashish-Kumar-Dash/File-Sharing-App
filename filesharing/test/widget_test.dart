import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:filesharing/main.dart';

void main() {
  testWidgets('App renders room entry screen', (WidgetTester tester) async {
    await tester.pumpWidget(const MyApp());
    await tester.pumpAndSettle();

    expect(find.text('Create Room'), findsOneWidget);
    expect(find.text('Join Room'), findsOneWidget);
    expect(find.text('File Share'), findsOneWidget);
  });
}
