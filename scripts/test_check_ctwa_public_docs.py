"""Synthetic, offline regression for the P0 webhook secret scanner."""
import importlib.util
from pathlib import Path
import tempfile
import unittest

src=Path(__file__).with_name("check_ctwa_public_docs.py")
spec=importlib.util.spec_from_file_location("ctwa_docs",src)
scanner=importlib.util.module_from_spec(spec)
spec.loader.exec_module(scanner)


class DocsSecretGateTests(unittest.TestCase):
    def test_full_hex_bearer_path_is_blocked(self):
        with tempfile.TemporaryDirectory() as work:
            root=Path(work)
            for path in scanner.DOCUMENTS:
                file=root/path
                file.parent.mkdir(parents=True,exist_ok=True)
                file.write_text("Documented event status only",encoding="utf-8")
            first=root/scanner.DOCUMENTS[0]
            first.write_text("URL /webhook/evolution/"+"a"*64,encoding="utf-8")
            self.assertEqual(scanner.check(root),[scanner.DOCUMENTS[0]])
            first.write_text("URL /webhook/ads/"+"b"*40,encoding="utf-8")
            self.assertEqual(scanner.check(root),[scanner.DOCUMENTS[0]])

    def test_redacted_docs_pass(self):
        with tempfile.TemporaryDirectory() as work:
            root=Path(work)
            for path in scanner.DOCUMENTS:
                file=root/path
                file.parent.mkdir(parents=True,exist_ok=True)
                file.write_text("/webhook/ads/<SEGREDO_PRIVADO_ROTACIONAR>",
                                encoding="utf-8")
            self.assertEqual(scanner.check(root),[])


if __name__=="__main__":
    unittest.main()
