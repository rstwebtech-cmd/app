import 'dart:async';
import 'dart:convert';
import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;
import 'package:google_sign_in/google_sign_in.dart';
import 'package:url_launcher/url_launcher.dart';

const base = String.fromEnvironment('API', defaultValue: 'http://10.0.2.2:8080');
const photo = '$base/static/janhavi.png';
const logo = '$base/static/logo.png';
String token = '';
Future buy(BuildContext c, String id) async {
  final r = await api('POST', '/api/pay/order', {'itemId': id});
  if (r['url'] == null) return snack(c, r['error'] ?? 'Payment error');
  await launchUrl(Uri.parse(r['url']), mode: LaunchMode.externalApplication);
}
Map me = {};
Future api(String m, String p, [Map? b]) async {
  final h = {'Content-Type': 'application/json', 'Authorization': 'Bearer $token'};
  final u = Uri.parse(base + p);
  final r = m == 'GET' ? await http.get(u, headers: h) : await http.post(u, headers: h, body: jsonEncode(b ?? {}));
  return jsonDecode(r.body);
}

void main() => runApp(MaterialApp(
    debugShowCheckedModeBanner: false,
    theme: ThemeData(colorSchemeSeed: const Color(0xFF6D28D9), scaffoldBackgroundColor: const Color(0xFFF5F3FF), useMaterial3: true),
    home: const Login()));

void go(BuildContext c, Widget w) => Navigator.push(c, MaterialPageRoute(builder: (_) => w));
void snack(BuildContext c, String s) => ScaffoldMessenger.of(c).showSnackBar(SnackBar(content: Text(s)));

class Login extends StatefulWidget { const Login({super.key}); @override State<Login> createState() => _L(); }
class _L extends State<Login> {
  final ph = TextEditingController(), otp = TextEditingController(), nm = TextEditingController();
  bool sent = false;
  void done(r) {
    if (r['token'] == null) return snack(context, r['error'] ?? 'Failed');
    token = r['token']; me = r['user'];
    Navigator.pushReplacement(context, MaterialPageRoute(builder: (_) => const Home()));
  }
  @override Widget build(BuildContext c) => Scaffold(
        body: Container(
          decoration: const BoxDecoration(gradient: LinearGradient(colors: [Color(0xFF2E1065), Color(0xFF7C3AED)], begin: Alignment.topCenter, end: Alignment.bottomCenter)),
          child: Center(child: Card(shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(28)), margin: const EdgeInsets.all(24), child: Padding(padding: const EdgeInsets.all(24), child: Column(mainAxisSize: MainAxisSize.min, children: [
            CircleAvatar(radius: 50, backgroundColor: const Color(0xFFEDE9FE), backgroundImage: const NetworkImage(photo), onBackgroundImageError: (_, __) {}), const SizedBox(height: 8),
            const Text('Janhavi Panwar', style: TextStyle(fontSize: 26, fontWeight: FontWeight.bold)),
            const Text('Wonder Girl of India'), const SizedBox(height: 16),
            TextField(controller: nm, decoration: const InputDecoration(labelText: 'Name')),
            TextField(controller: ph, keyboardType: TextInputType.phone, decoration: const InputDecoration(labelText: 'Mobile (+91...)')),
            if (sent) TextField(controller: otp, keyboardType: TextInputType.number, decoration: const InputDecoration(labelText: 'OTP')),
            const SizedBox(height: 12),
            FilledButton(onPressed: () async {
              if (!sent) { await api('POST', '/api/auth/otp/send', {'phone': ph.text}); setState(() => sent = true); }
              else { done(await api('POST', '/api/auth/otp/verify', {'phone': ph.text, 'code': otp.text, 'name': nm.text})); }
            }, child: Text(sent ? 'Verify OTP' : 'Send OTP')),
            OutlinedButton.icon(icon: const Icon(Icons.g_mobiledata, size: 30), label: const Text('Continue with Google'), onPressed: () async {
              final a = await (await GoogleSignIn(scopes: ['email']).signIn())?.authentication;
              if (a?.idToken != null) done(await api('POST', '/api/auth/google', {'idToken': a!.idToken}));
            }),
          ])))),
        ),
      );
}

class Home extends StatefulWidget { const Home({super.key}); @override State<Home> createState() => _H(); }
class _H extends State<Home> {
  int i = 0;
  @override Widget build(BuildContext c) {
    void jump(int v) => setState(() => i = v);
    final tabs = [HomeTab(jump), const Courses(), const QuizTab(), const Community(), Profile(jump)];
    return Scaffold(
      body: SafeArea(top: i != 0, child: tabs[i]),
      bottomNavigationBar: NavigationBar(selectedIndex: i, onDestinationSelected: jump, destinations: const [
        NavigationDestination(icon: Icon(Icons.home), label: 'Home'),
        NavigationDestination(icon: Icon(Icons.play_circle), label: 'Courses'),
        NavigationDestination(icon: Icon(Icons.emoji_events), label: 'Quiz'),
        NavigationDestination(icon: Icon(Icons.groups), label: 'Community'),
        NavigationDestination(icon: Icon(Icons.person), label: 'Profile'),
      ]));
  }
}
void page(BuildContext c, String t, Widget w) => go(c, Scaffold(appBar: AppBar(title: Text(t)), body: w));

class Pulse extends StatefulWidget { final Widget child; const Pulse(this.child, {super.key}); @override State<Pulse> createState() => _P(); }
class _P extends State<Pulse> with SingleTickerProviderStateMixin {
  late final a = AnimationController(vsync: this, duration: const Duration(milliseconds: 900))..repeat(reverse: true);
  @override void dispose() { a.dispose(); super.dispose(); }
  @override Widget build(BuildContext c) => ScaleTransition(scale: Tween(begin: 0.92, end: 1.08).animate(a), child: widget.child);
}
Widget cimg(Map k, double h) => ClipRRect(borderRadius: BorderRadius.circular(14), child: (k['Image'] ?? '').toString().isEmpty ? Container(height: h, width: double.infinity, color: const Color(0xFFDDD6FE), child: const Icon(Icons.menu_book, size: 36, color: Color(0xFF6D28D9))) : Image.network(base + k['Image'], height: h, width: double.infinity, fit: BoxFit.cover, errorBuilder: (_, __, ___) => Container(height: h, color: const Color(0xFFDDD6FE))));
Widget priceRow(num p, num mrp) => Wrap(crossAxisAlignment: WrapCrossAlignment.center, spacing: 4, children: [
      Text('₹$p', style: const TextStyle(fontSize: 20, fontWeight: FontWeight.bold, color: Colors.green)), const SizedBox(width: 8),
      Text('₹$mrp', style: const TextStyle(decoration: TextDecoration.lineThrough, color: Colors.grey)), const SizedBox(width: 8),
      Pulse(Container(padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2), decoration: BoxDecoration(color: Colors.red, borderRadius: BorderRadius.circular(8)),
          child: Text('${((mrp - p) * 100 / mrp).round()}% OFF', style: const TextStyle(color: Colors.white, fontSize: 12)))),
    ]);

Widget courseCard(BuildContext c, Map k) => InkWell(onTap: () => go(c, CourseScreen(k)), child: Card(clipBehavior: Clip.antiAlias, shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(18)), child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
      Stack(children: [cimg(k, 96), Positioned(top: 6, right: 6, child: Container(padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2), decoration: BoxDecoration(color: _blue, borderRadius: BorderRadius.circular(10)), child: Text(k['Category'], style: const TextStyle(color: Colors.white, fontSize: 11)))), const Positioned(bottom: 6, left: 6, child: CircleAvatar(radius: 14, backgroundColor: Colors.black54, child: Icon(Icons.play_arrow, color: Colors.white, size: 18)))]),
      Padding(padding: const EdgeInsets.all(8), child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Text(k['Title'], maxLines: 2, overflow: TextOverflow.ellipsis, style: const TextStyle(fontWeight: FontWeight.bold)),
        Row(children: [const Icon(Icons.access_time, size: 13), Text(' ${k['Duration']}', style: const TextStyle(fontSize: 12))]),
        priceRow(k['Price'], k['MRP']),
        SizedBox(width: double.infinity, child: FilledButton(style: FilledButton.styleFrom(backgroundColor: _blue), onPressed: () => go(c, CourseScreen(k)), child: const Text('Enroll Now')))]))])));

class Courses extends StatefulWidget { final String q; const Courses({super.key, this.q = ''}); @override State<Courses> createState() => _Cs(); }
class _Cs extends State<Courses> {
  String cat = 'All';
  static const cats = ['All', 'English', 'Accent', 'Languages', 'Career'];
  @override Widget build(BuildContext c) => FutureBuilder(future: api('GET', '/api/courses'), builder: (_, s) {
        if (!s.hasData) return const Center(child: CircularProgressIndicator());
        final l = (s.data as List).where((x) => (cat == 'All' || x['Category'] == cat) && x['Title'].toString().toLowerCase().contains(widget.q.toLowerCase())).toList();
        return Column(children: [
          SizedBox(height: 56, child: ListView(scrollDirection: Axis.horizontal, padding: const EdgeInsets.all(8), children: [for (final t in cats) Padding(padding: const EdgeInsets.only(right: 8), child: ChoiceChip(label: Text(t), selected: cat == t, onSelected: (_) => setState(() => cat = t)))])),
          Expanded(child: GridView.count(crossAxisCount: 2, childAspectRatio: .62, crossAxisSpacing: 10, mainAxisSpacing: 10, padding: const EdgeInsets.all(10), children: [for (final k in l) courseCard(c, k)]))]);
      });
}

class CourseScreen extends StatefulWidget { final Map k; const CourseScreen(this.k, {super.key}); @override State<CourseScreen> createState() => _C(); }
class _C extends State<CourseScreen> {
  final rv = TextEditingController();
  @override Widget build(BuildContext c) {
    final k = widget.k;
    return Scaffold(appBar: AppBar(title: Text(k['Title'])), body: ListView(padding: const EdgeInsets.all(16), children: [
      cimg(k, 190), const SizedBox(height: 10), Text('${k['Category']} · ${k['Duration']}'), priceRow(k['Price'], k['MRP']), const SizedBox(height: 12),
      FilledButton(onPressed: () => buy(c, k['ID']), child: Text('Buy Now ₹${k['Price']}')),
      OutlinedButton(onPressed: () => go(c, Quiz(k)), child: const Text('Take Final Quiz → Get Certificate')),
      const Divider(), const Text('Reviews', style: TextStyle(fontWeight: FontWeight.bold)),
      FutureBuilder(future: api('GET', '/api/reviews'), builder: (_, s) => Column(children: [
        for (final r in (s.data as List? ?? [])) ListTile(title: Text(r['Name']), subtitle: Text(r['Text']), trailing: Text('⭐${r['Stars']}'))])),
      TextField(controller: rv, decoration: const InputDecoration(labelText: 'Write a review (shown after approval)')),
      TextButton(onPressed: () async { await api('POST', '/api/reviews', {'courseId': k['ID'], 'text': rv.text, 'stars': 5}); rv.clear(); snack(c, 'Thanks! Review pending approval'); }, child: const Text('Submit review')),
    ]));
  }
}

class Combos extends StatelessWidget {
  const Combos({super.key});
  @override Widget build(BuildContext c) => FutureBuilder(future: api('GET', '/api/combos'), builder: (_, s) {
        if (!s.hasData) return const Center(child: CircularProgressIndicator());
        return ListView(children: [for (final b in s.data as List) Card(color: Colors.amber.shade50, margin: const EdgeInsets.all(12), child: Padding(padding: const EdgeInsets.all(16), child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          const Text('🔥 SPECIAL COMBO OFFER'), Text(b['Title'], style: const TextStyle(fontSize: 20, fontWeight: FontWeight.bold)), priceRow(b['Price'], b['MRP']),
          FilledButton(onPressed: () => buy(c, b['ID']), child: const Text('Get Offer'))])))]);
      });
}

class Community extends StatelessWidget {
  const Community({super.key});
  @override Widget build(BuildContext c) => FutureBuilder(future: api('GET', '/api/groups'), builder: (_, s) => Scaffold(
        floatingActionButton: FloatingActionButton(child: const Icon(Icons.add), onPressed: () async {
          await api('POST', '/api/groups', {'name': 'Group ${DateTime.now().minute}${DateTime.now().second}'}); snack(c, 'Group created - reopen tab'); }),
        body: ListView(children: [for (final g in (s.data as List? ?? [])) ListTile(leading: const Icon(Icons.groups), title: Text(g['Name']), onTap: () => go(c, Chat(room: g['ID'], title: g['Name'])))])));
}

class Chat extends StatefulWidget { final String room, title; const Chat({super.key, required this.room, required this.title}); @override State<Chat> createState() => _Ch(); }
class _Ch extends State<Chat> {
  final t = TextEditingController(); List m = []; Timer? tm;
  Future poll() async { final r = await api('GET', '/api/chat?room=${widget.room}&after=0'); if (mounted) setState(() => m = r); }
  @override void initState() { super.initState(); poll(); tm = Timer.periodic(const Duration(seconds: 3), (_) => poll()); }
  @override void dispose() { tm?.cancel(); super.dispose(); }
  @override Widget build(BuildContext c) => Scaffold(appBar: AppBar(title: Text(widget.title)), body: Column(children: [
        Expanded(child: ListView(children: [for (final x in m) ListTile(dense: true, title: Text(x['Name'], style: const TextStyle(fontWeight: FontWeight.bold)), subtitle: Text(x['Text']))])),
        Row(children: [Expanded(child: Padding(padding: const EdgeInsets.all(8), child: TextField(controller: t))), IconButton(icon: const Icon(Icons.send), onPressed: () async { await api('POST', '/api/chat', {'room': widget.room, 'text': t.text}); t.clear(); poll(); })])]));
}

class Quiz extends StatefulWidget { final Map k; const Quiz(this.k, {super.key}); @override State<Quiz> createState() => _Q(); }
class _Q extends State<Quiz> {
  List qs = []; Map<int, int> a = {};
  @override void initState() { super.initState(); api('GET', '/api/quiz/${widget.k['ID']}').then((v) => setState(() => qs = v)); }
  @override Widget build(BuildContext c) => Scaffold(appBar: AppBar(title: const Text('Final Quiz')), body: ListView(padding: const EdgeInsets.all(12), children: [
        for (var i = 0; i < qs.length; i++) Card(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Padding(padding: const EdgeInsets.all(12), child: Text('${i + 1}. ${qs[i]['Q']}', style: const TextStyle(fontWeight: FontWeight.bold))),
          for (var j = 0; j < qs[i]['Options'].length; j++) RadioListTile<int>(value: j, groupValue: a[i], title: Text(qs[i]['Options'][j]), onChanged: (v) => setState(() => a[i] = v!))])),
        FilledButton(onPressed: () async {
          final r = await api('POST', '/api/quiz/${widget.k['ID']}', {'answers': [for (var i = 0; i < qs.length; i++) a[i] ?? -1]});
          if (!mounted) return;
          showDialog(context: c, builder: (_) => AlertDialog(title: Text(r['passed'] ? '🎉 Passed ${r['score']}/${r['total']}' : 'Score ${r['score']}/${r['total']} - try again'),
              content: Text(r['passed'] ? 'Certificate: $base/certificate/${r['certificate']}' : 'Need 60% to pass')));
        }, child: const Text('Submit')),
      ]));
}

const _blue = Color(0xFF6D28D9);
const _dark = Color(0xFF2E1065);
const _gold = Color(0xFFFBBF24);

class HomeTab extends StatelessWidget {
  final void Function(int) jump;
  const HomeTab(this.jump, {super.key});
  Widget head(String t, [VoidCallback? f]) => Padding(padding: const EdgeInsets.fromLTRB(16, 18, 16, 8), child: Row(children: [Container(width: 4, height: 22, color: _blue), const SizedBox(width: 8), Expanded(child: Text(t, style: const TextStyle(fontSize: 19, fontWeight: FontWeight.bold))), if (f != null) TextButton(onPressed: f, child: const Text('View All →'))]));
  Widget ico(IconData i, String t, Color col, VoidCallback f) => InkWell(onTap: f, child: Container(width: 76, margin: const EdgeInsets.all(4), padding: const EdgeInsets.symmetric(vertical: 10), decoration: BoxDecoration(color: col.withOpacity(.12), borderRadius: BorderRadius.circular(16)), child: Column(children: [Icon(i, color: col, size: 28), const SizedBox(height: 4), Text(t, style: const TextStyle(fontSize: 11), textAlign: TextAlign.center)])));
  Widget qc(IconData i, String t, String s, Color col, VoidCallback f) => Expanded(child: InkWell(onTap: f, child: Container(margin: const EdgeInsets.all(4), padding: const EdgeInsets.symmetric(vertical: 12, horizontal: 4), decoration: BoxDecoration(color: col.withOpacity(.12), borderRadius: BorderRadius.circular(16)), child: Column(children: [Icon(i, color: col), Text(t, style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 12), textAlign: TextAlign.center), Text(s, style: const TextStyle(fontSize: 10), textAlign: TextAlign.center)]))));
  @override Widget build(BuildContext c) => FutureBuilder(future: Future.wait([api('GET', '/api/me'), api('GET', '/api/courses'), api('GET', '/api/live'), api('GET', '/api/reviews')]), builder: (_, s) {
        if (!s.hasData) return const Center(child: CircularProgressIndicator());
        final d = s.data as List; final u = d[0]['user']; final courses = d[1] as List; final live = (d[2] as List? ?? []); final revs = (d[3] as List? ?? []);
        final nm = (u['Name'] ?? '').toString().isEmpty ? 'Student' : u['Name'];
        return ListView(padding: EdgeInsets.zero, children: [
          Container(height: 270, decoration: const BoxDecoration(gradient: LinearGradient(colors: [_dark, Color(0xFF7C3AED)], begin: Alignment.topLeft, end: Alignment.bottomRight), borderRadius: BorderRadius.vertical(bottom: Radius.circular(28))), child: Stack(children: [
            Positioned(right: 0, bottom: 0, child: Image.network(photo, height: 240, fit: BoxFit.contain, errorBuilder: (_, __, ___) => const SizedBox())),
            SafeArea(child: Padding(padding: const EdgeInsets.fromLTRB(16, 8, 140, 12), child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Image.network(logo, height: 34, errorBuilder: (_, __, ___) => const Text('Janhavi Panwar', style: TextStyle(color: Colors.white, fontSize: 20))),
              const Spacer(),
              const Text('Dream Big...', style: TextStyle(color: Colors.white, fontSize: 22, fontWeight: FontWeight.bold)),
              const Text('Learn Beyond Limits!', style: TextStyle(color: _gold, fontSize: 22, fontWeight: FontWeight.bold)),
              Text('Hello, $nm 👋 · v3', style: const TextStyle(color: Colors.white70)), const SizedBox(height: 8),
              FilledButton(style: FilledButton.styleFrom(backgroundColor: _gold, foregroundColor: Colors.black), onPressed: () => jump(1), child: const Text('Explore Courses →'))])))])),
          Padding(padding: const EdgeInsets.all(14), child: TextField(decoration: InputDecoration(hintText: 'Search courses, quizzes, or topics...', prefixIcon: const Icon(Icons.search), filled: true, fillColor: Colors.white, border: OutlineInputBorder(borderRadius: BorderRadius.circular(30), borderSide: BorderSide.none)), onSubmitted: (v) => page(c, 'Search: $v', Courses(q: v)))),
          SingleChildScrollView(scrollDirection: Axis.horizontal, padding: const EdgeInsets.symmetric(horizontal: 8), child: Row(children: [
            ico(Icons.school, 'Courses', _blue, () => jump(1)), ico(Icons.star, 'Reviews', Colors.orange, () => page(c, 'Reviews', const ReviewsPage())),
            ico(Icons.chat, 'Live Chat', Colors.teal, () => go(c, const Chat(room: 'support', title: 'Live Support'))), ico(Icons.groups, 'Community', Colors.purple, () => jump(3)),
            ico(Icons.quiz, 'Quiz', Colors.indigo, () => jump(2)), ico(Icons.bar_chart, 'Level Check', Colors.green, () => jump(4))])),
          head('Featured Courses', () => jump(1)),
          GridView.count(crossAxisCount: 2, shrinkWrap: true, physics: const NeverScrollableScrollPhysics(), childAspectRatio: .62, crossAxisSpacing: 10, mainAxisSpacing: 10, padding: const EdgeInsets.symmetric(horizontal: 12), children: [for (final k in courses.take(4)) courseCard(c, k)]),
          Container(margin: const EdgeInsets.fromLTRB(14, 18, 14, 4), padding: const EdgeInsets.all(18), decoration: BoxDecoration(gradient: const LinearGradient(colors: [_dark, Color(0xFF7C3AED)]), borderRadius: BorderRadius.circular(22)), child: Row(children: [
            Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Container(padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2), decoration: BoxDecoration(color: Colors.red, borderRadius: BorderRadius.circular(8)), child: Text(live.isEmpty ? 'Coming Soon' : 'Live', style: const TextStyle(color: Colors.white, fontSize: 12))),
              Text(live.isEmpty ? 'Live Classes' : live[0]['Title'], style: const TextStyle(color: Colors.white, fontSize: 20, fontWeight: FontWeight.bold)),
              Text(live.isEmpty ? 'Jald hi schedule hogi' : live[0]['At'], style: const TextStyle(color: Colors.white70)), const SizedBox(height: 8),
              FilledButton(style: FilledButton.styleFrom(backgroundColor: Colors.white, foregroundColor: _blue), onPressed: () => live.isEmpty ? page(c, 'Live Classes', const LiveTab()) : launchUrl(Uri.parse(live[0]['Link']), mode: LaunchMode.externalApplication), child: const Text('Join Live →'))])),
            const Icon(Icons.podcasts, color: _gold, size: 60)])),
          Padding(padding: const EdgeInsets.fromLTRB(10, 12, 10, 0), child: Row(children: [
            qc(Icons.local_offer, 'Combo Offers', 'Save more', Colors.deepOrange, () => page(c, 'Combo Offers', const Combos())), qc(Icons.trending_up, 'Your Progress', 'Track Level', Colors.orange, () => jump(4)),
            qc(Icons.workspace_premium, 'Certificates', 'Get Certified', Colors.amber, () => jump(4)), qc(Icons.quiz, 'Take Quiz', 'Test Skills', Colors.indigo, () => jump(2))])),
          head('What Our Learners Say', () => page(c, 'Reviews', const ReviewsPage())),
          Padding(padding: const EdgeInsets.symmetric(horizontal: 14), child: Row(children: [
            Expanded(flex: 2, child: Container(padding: const EdgeInsets.all(14), decoration: BoxDecoration(color: Colors.white, borderRadius: BorderRadius.circular(16)), child: revs.isEmpty ? const Text('Pehle review aap likho! ⭐') : Column(crossAxisAlignment: CrossAxisAlignment.start, children: [Text('⭐' * (revs[0]['Stars'] as int)), Text('"${revs[0]['Text']}"', maxLines: 3, overflow: TextOverflow.ellipsis), Text('— ${revs[0]['Name']}', style: const TextStyle(fontSize: 12, color: Colors.grey))]))),
            const SizedBox(width: 10),
            Expanded(child: Container(padding: const EdgeInsets.all(14), decoration: BoxDecoration(color: const Color(0xFFEDE9FE), borderRadius: BorderRadius.circular(16)), child: const Column(children: [Icon(Icons.groups, color: _blue), Text('2.27M+', style: TextStyle(fontSize: 20, fontWeight: FontWeight.bold)), Text('YouTube Family', style: TextStyle(fontSize: 11), textAlign: TextAlign.center)])))])),
          const SizedBox(height: 24)]);
      });
}

class ReviewsPage extends StatefulWidget { const ReviewsPage({super.key}); @override State<ReviewsPage> createState() => _Rv(); }
class _Rv extends State<ReviewsPage> {
  final t = TextEditingController();
  @override Widget build(BuildContext c) => FutureBuilder(future: api('GET', '/api/reviews'), builder: (_, s) => ListView(padding: const EdgeInsets.all(12), children: [
        TextField(controller: t, decoration: const InputDecoration(labelText: 'Apna review likho (approval ke baad dikhega)')),
        TextButton(onPressed: () async { await api('POST', '/api/reviews', {'text': t.text, 'stars': 5}); t.clear(); snack(c, 'Thanks! Review pending approval'); }, child: const Text('Submit review')),
        for (final r in (s.data as List? ?? [])) Card(child: ListTile(title: Text(r['Name']), subtitle: Text(r['Text']), trailing: Text('⭐${r['Stars']}')))]));
}

class LiveTab extends StatelessWidget {
  const LiveTab({super.key});
  @override Widget build(BuildContext c) => FutureBuilder(future: api('GET', '/api/live'), builder: (_, s) {
        if (!s.hasData) return const Center(child: CircularProgressIndicator());
        final l = s.data as List;
        if (l.isEmpty) return const Center(child: Text('Koi live class scheduled nahi'));
        return ListView(children: [for (final x in l) Card(margin: const EdgeInsets.all(10), child: ListTile(leading: const Icon(Icons.videocam, color: _blue), title: Text(x['Title']), subtitle: Text(x['At']), trailing: FilledButton(onPressed: () => launchUrl(Uri.parse(x['Link']), mode: LaunchMode.externalApplication), child: const Text('Join Now'))))]);
      });
}

class QuizTab extends StatelessWidget {
  const QuizTab({super.key});
  @override Widget build(BuildContext c) => FutureBuilder(future: api('GET', '/api/courses'), builder: (_, s) {
        if (!s.hasData) return const Center(child: CircularProgressIndicator());
        return ListView(children: [const Padding(padding: EdgeInsets.all(14), child: Text('Quiz do aur certificate pao 🏆', style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold))),
          for (final k in s.data as List) Card(margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 4), child: ListTile(leading: SizedBox(width: 80, child: cimg(k, 50)), title: Text(k['Title']), trailing: const Icon(Icons.chevron_right), onTap: () => go(c, Quiz(k))))]);
      });
}

class Profile extends StatelessWidget {
  final void Function(int) jump;
  const Profile(this.jump, {super.key});
  @override Widget build(BuildContext c) => FutureBuilder(future: api('GET', '/api/me'), builder: (_, s) {
        if (!s.hasData) return const Center(child: CircularProgressIndicator());
        final d = s.data as Map; final u = d['user']; final certs = (d['certs'] as List? ?? []);
        final nm = (u['Name'] ?? '').toString(); final ini = nm.isEmpty ? 'S' : nm[0].toUpperCase();
        final enr = ((u['Courses'] as List? ?? [])).toSet().length;
        final pts = enr * 100 + certs.length * 500; final lvl = 1 + pts ~/ 500; final pct = (pts % 500) / 500;
        final lname = lvl < 3 ? 'Beginner' : lvl < 5 ? 'Intermediate' : 'Advanced Learner';
        Widget stat(IconData i, String v, String t) => Expanded(child: Column(children: [Icon(i, color: _blue), Text(v, style: const TextStyle(fontSize: 18, fontWeight: FontWeight.bold)), Text(t, style: const TextStyle(fontSize: 11), textAlign: TextAlign.center)]));
        Widget item(IconData i, String t, String sub, VoidCallback f, [Color? col]) => Container(margin: const EdgeInsets.symmetric(horizontal: 14, vertical: 4), decoration: BoxDecoration(color: Colors.white, borderRadius: BorderRadius.circular(14), border: Border.all(color: const Color(0xFFDDD6FE))), child: ListTile(leading: Icon(i, color: col ?? _blue, size: 30), title: Text(t, style: const TextStyle(fontWeight: FontWeight.bold)), subtitle: Text(sub), trailing: Icon(Icons.chevron_right, color: col ?? _blue), onTap: f));
        Widget badge(IconData i, String t, bool on) => Expanded(child: Container(margin: const EdgeInsets.all(3), padding: const EdgeInsets.symmetric(vertical: 10), decoration: BoxDecoration(color: Colors.white, borderRadius: BorderRadius.circular(12)), child: Column(children: [Icon(i, size: 30, color: on ? _gold : Colors.grey.shade400), Text(t, style: const TextStyle(fontSize: 10), textAlign: TextAlign.center)])));
        return ListView(padding: EdgeInsets.zero, children: [
          Container(padding: const EdgeInsets.fromLTRB(16, 40, 16, 40), decoration: const BoxDecoration(gradient: LinearGradient(colors: [_dark, Color(0xFF7C3AED)]), borderRadius: BorderRadius.vertical(bottom: Radius.circular(32))), child: Row(children: [
            CircleAvatar(radius: 46, backgroundColor: _gold, child: CircleAvatar(radius: 42, backgroundColor: Colors.white, child: Text(ini, style: const TextStyle(fontSize: 36, color: _blue)))), const SizedBox(width: 14),
            Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Text(nm.isEmpty ? 'Student' : nm, style: const TextStyle(fontSize: 22, fontWeight: FontWeight.bold, color: Colors.white)),
              Text('${u['Email'] ?? ''}${u['Phone'] ?? ''}', style: const TextStyle(color: Colors.white70)), const SizedBox(height: 6),
              Container(padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4), decoration: BoxDecoration(color: _blue, borderRadius: BorderRadius.circular(20), border: Border.all(color: _gold)), child: Text(enr > 0 ? '💎 Premium Member' : 'Free Member', style: const TextStyle(color: Colors.white, fontSize: 12)))]))])),
          Container(margin: const EdgeInsets.all(14), padding: const EdgeInsets.symmetric(vertical: 14), decoration: BoxDecoration(color: Colors.white, borderRadius: BorderRadius.circular(18)), child: Row(children: [stat(Icons.play_circle, '$enr', 'Courses Enrolled'), stat(Icons.military_tech, '$lvl', 'Level'), stat(Icons.star, '$pts', 'Points'), stat(Icons.workspace_premium, '${certs.length}', 'Certificates')])),
          Container(margin: const EdgeInsets.symmetric(horizontal: 14), padding: const EdgeInsets.all(16), decoration: BoxDecoration(gradient: const LinearGradient(colors: [_dark, Color(0xFF7C3AED)]), borderRadius: BorderRadius.circular(20)), child: Row(children: [
            SizedBox(width: 90, height: 90, child: Stack(alignment: Alignment.center, children: [SizedBox(width: 90, height: 90, child: CircularProgressIndicator(value: pct == 0 ? 0.02 : pct, strokeWidth: 9, color: _gold, backgroundColor: Colors.white24)), Text('${(pct * 100).round()}%', style: const TextStyle(color: Colors.white, fontSize: 20, fontWeight: FontWeight.bold))])), const SizedBox(width: 16),
            Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [const Text('Your Learning Progress', style: TextStyle(color: Colors.white70, fontSize: 12)), Text('👑 Level $lvl', style: const TextStyle(color: Colors.white, fontSize: 22, fontWeight: FontWeight.bold)), Text(lname, style: const TextStyle(color: Colors.white70)), const SizedBox(height: 6), LinearProgressIndicator(value: pct, color: _gold, backgroundColor: Colors.white24), const SizedBox(height: 4), Text('⭐ $pts Points', style: const TextStyle(color: Colors.white))])),
            const Icon(Icons.emoji_events, color: _gold, size: 54)])),
          const SizedBox(height: 10),
          item(Icons.play_circle, 'My Courses', 'Continue your learning journey', () => jump(1)),
          item(Icons.quiz, 'My Quizzes', 'Check your quiz performance', () => jump(2)),
          item(Icons.workspace_premium, 'My Certificates', 'Download your certificates', () => page(c, 'My Certificates', ListView(children: [for (final x in certs) ListTile(leading: const Icon(Icons.workspace_premium, color: _gold), title: Text(x['CourseTitle']), subtitle: Text(x['Date']), onTap: () => launchUrl(Uri.parse('$base/certificate/${x['ID']}'), mode: LaunchMode.externalApplication))]))),
          item(Icons.star, 'My Reviews', 'Share your feedback', () => page(c, 'Reviews', const ReviewsPage())),
          item(Icons.groups, 'My Community', 'Connect with learners', () => jump(3)),
          item(Icons.chat, 'Live Chat', 'Chat with our support team', () => go(c, const Chat(room: 'support', title: 'Live Support'))),
          item(Icons.logout, 'Logout', 'Sign out from your account', () { token = ''; Navigator.pushAndRemoveUntil(c, MaterialPageRoute(builder: (_) => const Login()), (_) => false); }, Colors.red),
          Container(margin: const EdgeInsets.all(14), padding: const EdgeInsets.all(12), decoration: BoxDecoration(color: const Color(0xFFEDE9FE), borderRadius: BorderRadius.circular(18)), child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
            const Text('⭐ Your Achievements', style: TextStyle(fontWeight: FontWeight.bold, fontSize: 16)), const SizedBox(height: 8),
            Row(children: [badge(Icons.school, 'First Course', enr >= 1), badge(Icons.verified, 'Quiz Master', certs.isNotEmpty), badge(Icons.emoji_events, 'Course Champion', certs.length >= 3), badge(Icons.military_tech, 'Level 3+', lvl >= 3)])]))]);
      });
}
