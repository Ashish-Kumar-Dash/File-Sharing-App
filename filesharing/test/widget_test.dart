import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:filesharing/main.dart';

void main() {
  testWidgets('App renders upload and download buttons', (WidgetTester tester) async {
    await tester.pumpWidget(const MyApp());
    await tester.pumpAndSettle();

    expect(find.text('Upload'), findsOneWidget);
    expect(find.text('Download'), findsOneWidget);
    expect(find.text('File Share'), findsOneWidget);
  });
}
