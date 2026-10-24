using System;
using System.Collections.Generic;
using System.IO;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Threading;

partial class SettingsApp {
    [System.Runtime.InteropServices.DllImport("user32.dll")]static extern bool IsWindowEnabled(IntPtr handle);
    static void Check(bool ok,string reason){if(!ok)throw new Exception(reason);}
    static T InDialog<T>(Window d,string name) where T:class{return LogicalTreeHelper.FindLogicalNode(d,name) as T;}
    static void RunSelfTests(string path){
        var results=new List<string>();
        window.Opacity=0;window.ShowActivated=false;window.WindowStartupLocation=WindowStartupLocation.Manual;window.Left=-30000;window.Top=-30000;window.Show();
        Get<ComboBox>("Output").ItemsSource=new[]{new OutputDevice{ID="saved-output",Name="Headphones (Arctis Nova Pro Wireless) — long device name"}};
        Get<ComboBox>("Output").SelectedIndex=0;Get<CheckBox>("Redirect").IsChecked=true;
        Get<CheckBox>("Echo").IsChecked=false;Get<CheckBox>("Disable").IsChecked=false;Get<CheckBox>("Startup").IsChecked=false;
        running=true;echoSelected=true;echoState="blocked";deviceState="disabled";
        foreach(bool initialArabic in new[]{false,true}){
            Get<CheckBox>("Disable").IsChecked=false;arabic=initialArabic;ApplyLanguage();
            foreach(int pass in new[]{0,1,2,3}){
                bool before=arabic;Get<Button>("LanguageToggle").RaiseEvent(new RoutedEventArgs(Button.ClickEvent));
                Check(arabic!=before,"Language icon callback");
                Check(Get<StackPanel>("Root").FlowDirection==(arabic?FlowDirection.RightToLeft:FlowDirection.LeftToRight),"RTL/LTR");
                Check(!Is("Echo")&&!Is("Disable")&&!Is("Startup")&&Is("Redirect")&&running,"Language changed feature state");
                Check(((OutputDevice)Get<ComboBox>("Output").SelectedItem).ID=="saved-output","Language lost output");
                foreach(var name in new[]{"Save","Stop","Restore","Uninstall","About","Logs","Diagnostics","Echo","Disable","Startup","Redirect"})Check(!String.IsNullOrEmpty((string)Get<ContentControl>(name).Content),"Missing label "+name);
            }
            foreach(var kind in new[]{"uninstall","restore","about"}){
                var d=BuildConfirmation(kind);var title=InDialog<TextBlock>(d,"DialogTitle").Text;
                InDialog<Button>(d,"DialogLanguage").RaiseEvent(new RoutedEventArgs(Button.ClickEvent));
                Check(InDialog<TextBlock>(d,"DialogTitle").Text!=title,"Dialog did not relocalize");
                Check(d.FlowDirection==Get<StackPanel>("Root").FlowDirection,"Dialog direction");
                Check(InDialog<Button>(d,"Confirm").FocusVisualStyle==null,"Dotted dialog focus");
                if(kind=="about"){Check(InDialog<TextBlock>(d,"Question").Text==S("current_version")+": "+ProductVersion,"About version source");Check(InDialog<Button>(d,"SaveRepository")==null&&InDialog<Button>(d,"OpenRepository")!=null&&InDialog<Button>(d,"CheckUpdates")!=null,"About update controls");Check(EffectiveRepository=="tuxp1/SonarFix","Official update repository is fixed");}
                else {
                    Check(InDialog<Button>(d,"Cancel").IsCancel&&InDialog<Button>(d,"Cancel").IsDefault,"Safe modal default");
                    // Exercise actual modal cancellation without showing a test window.
                    d.Opacity=0;d.ShowActivated=false;d.WindowStartupLocation=WindowStartupLocation.Manual;d.Left=-30000;d.Top=-30000;
                    d.Loaded+=(s,e)=>d.Dispatcher.BeginInvoke(new Action(()=>{Check(d.Owner==window&&!IsWindowEnabled(new System.Windows.Interop.WindowInteropHelper(window).Handle),"Modal owner remains interactive");InDialog<Button>(d,"Cancel").RaiseEvent(new RoutedEventArgs(Button.ClickEvent));}));
                    Check(d.ShowDialog()!=true,"Cancel accepted a destructive operation");
                    Check(!uninstalling&&running&&window.IsVisible&&IsWindowEnabled(new System.Windows.Interop.WindowInteropHelper(window).Handle),"Cancel changed worker/main window state");
                }
            }
            foreach(double scale in new[]{1.0,1.25,1.5})foreach(bool expanded in new[]{false,true}){
                Get<Expander>("Advanced").IsExpanded=expanded;Get<CheckBox>("Disable").IsChecked=true;ApplyLanguage();
                var shell=Get<DockPanel>("Shell");double height=Math.Min(640,1020/scale-70);
                shell.Measure(new Size(730,height));shell.Arrange(new Rect(0,0,730,height));shell.UpdateLayout();
                foreach(var name in new[]{"Save","Stop","LanguageToggle","Echo","Disable","Output"}){
                    var el=Get<FrameworkElement>(name);Check(el.ActualWidth>0&&el.ActualHeight>0,"Layout "+name);
                }
                Check(Get<Button>("Save").ActualWidth+Get<Button>("Stop").ActualWidth<730,"Footer overlap");
                if(expanded)foreach(var name in new[]{"Restore","About","Uninstall","OpenFolder","CopyFolder","Location"})Check(Get<FrameworkElement>(name).ActualWidth>0,"Advanced layout "+name);
                results.Add((arabic?"Arabic":"English")+" logical DPI "+scale+" advanced="+expanded+": PASS");
            }
        }
        // Save follows unsaved audio changes only; language switches are not changes.
        savedState=CurrentState();UpdateSave();Check(!Get<Button>("Save").IsEnabled,"Save enabled without changes");
        Get<CheckBox>("Echo").IsChecked=!Is("Echo");Check(Get<Button>("Save").IsEnabled,"Save not enabled after a change");
        Get<CheckBox>("Echo").IsChecked=!Is("Echo");Check(!Get<Button>("Save").IsEnabled,"Save stays enabled after reverting");
        Get<CheckBox>("Startup").IsChecked=!Is("Startup");Get<Button>("LanguageToggle").RaiseEvent(new RoutedEventArgs(Button.ClickEvent));Check(Get<Button>("Save").IsEnabled,"Language switch lost the unsaved change");
        Get<CheckBox>("Startup").IsChecked=!Is("Startup");Get<Button>("LanguageToggle").RaiseEvent(new RoutedEventArgs(Button.ClickEvent));Check(!Get<Button>("Save").IsEnabled,"Language switch counted as a change");
        Check(DockPanel.GetDock(Get<Button>("LanguageToggle"))==Dock.Top&&Get<Button>("LanguageToggle").Parent==Get<DockPanel>("Shell"),"Language button belongs to the window header, not the status card");
        echoState="device_unavailable";running=true;echoSelected=true;ApplyLanguage();Check(Get<TextBlock>("Status").Text==S("status_device_unavailable"),"Device unavailable status");
        echoState="blocked";ApplyLanguage();Check(Get<TextBlock>("Status").Text==S("status_4"),"Blocked status");
        Check(!Get<DockPanel>("Shell").Focusable&&!Get<StackPanel>("Root").Focusable&&!Get<ScrollViewer>("MainScroll").Focusable,"Container focus");
        Check(Get<Button>("Save").Focusable&&Get<Button>("Save").FocusVisualStyle==null,"Keyboard focus styling");
        Check(Get<TextBox>("Location").IsReadOnly&&Get<TextBox>("Location").FlowDirection==FlowDirection.LeftToRight&&Get<TextBox>("Location").Text==directory,"Installation folder");
        Check(window.FindName("Footer")==null&&window.FindName("Subtitle")==null,"Main screen version/title");
        Check(Get<DockPanel>("Shell").MaxWidth==760,"Content width cap");
        Check(window.FindName("DiagnosticsSection")==null&&Get<System.Windows.Controls.Primitives.UniformGrid>("AudioActions").Children.Count==4,"Audio & Diagnostics grouping");
        Check(Get<Button>("Uninstall").HorizontalAlignment==HorizontalAlignment.Left,"Compact uninstall action");
        Check(UninstallArguments.StartsWith("/SILENT /NORESTART /WAITPID="),"Uninstall confirmation duplication");
        results.Add("Language icon, modal relocalization/cancel, readonly folder, focus, grouped advanced, About version, uninstall arguments: PASS");
        RunFeatureTests(results);window.Close();File.WriteAllText(path,json.Serialize(results));
    }
}




