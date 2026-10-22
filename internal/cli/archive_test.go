package cli_test

import (
 "bytes"
 "testing"

 "github.com/daiksud/gh-settings/internal/cli"
)

func TestArchiveExportFlagIsOptInAndComposable(t *testing.T) {
 for _, tc := range []struct{args []string;valid bool;want cli.ExportOptions}{
  {[]string{"export"},true,cli.ExportOptions{}},
  {[]string{"export","--archive"},true,cli.ExportOptions{Archive:true}},
  {[]string{"export","--archive","--rulesets","--retention"},true,cli.ExportOptions{Archive:true,Rulesets:true,Retention:true}},
  {[]string{"export","--archive","--archive"},false,cli.ExportOptions{}},
  {[]string{"plan","--archive"},false,cli.ExportOptions{}},
 }{
  called:=false
  var got cli.ExportOptions
  var out,errOut bytes.Buffer
  status:=cli.Run(tc.args,&out,&errOut,"dev",cli.Operations{Export:func(o cli.ExportOptions)error{called=true;got=o;return nil}})
  if (status==0)!=tc.valid || called!=tc.valid || (tc.valid && got!=tc.want){t.Fatal(tc.args,status,called,got,errOut.String())}
 }
}
