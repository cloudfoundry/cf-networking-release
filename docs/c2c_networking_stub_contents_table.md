<table border="1" class="nice">
  <tr>
    <th style="width:35%">Container-to-Container Networking Opsfiles</th>
    <th>Description</th>
  </tr>
  <tr>
    <td><pre><code>
    <!-- code_snippet c2c_networking_stub_table_r1c1 start -->
- type: replace
  path: /instance_groups/name=diego-cell/jobs/name=vxlan_policy_agent/properties/iptables_logging?
  value: true
    <!-- code_snippet c2c_networking_stub_table_r1c1 end -->
    </code></pre></td>
    <td>
      <!-- code_snippet c2c_networking_stub_table_r1c2 start -->
      The default value for <code>iptables_logging</code> is <code>false</code>.
    <br><br>
    (Optional) Change the value to <code>true</code> to enable logging for Container-to-Container policy iptables rules.
    <!-- code_snippet c2c_networking_stub_table_r1c2 end -->
    </td>
  </tr>
  <tr>
    <td><pre><code>
    <!-- code_snippet c2c_networking_stub_table_r2c1 start -->
- type: replace
  path: /instance_groups/name=diego-cell/jobs/name=cni/properties/iptables_logging?
  value: true
    <!-- code_snippet c2c_networking_stub_table_r2c1 end -->
    </code></pre></td>
    <td>
    <!-- code_snippet c2c_networking_stub_table_r2c2 start -->
      The default value for <code>iptables_logging</code> is <code>false</code>.
    <br><br>
    (Optional) Change the value to <code>true</code> to enable
    logging for Application Security Group (ASG) iptables rules.
    <!-- code_snippet c2c_networking_stub_table_r2c2 end -->
    </td>
  </tr>
  <tr>
    <td><pre><code>
    <!-- code_snippet c2c_networking_stub_table_r3c1 start -->
- type: replace
  path: /instance_groups/name=diego-cell/jobs/name=silk-controller/properties/network?
  value: REPLACE-WITH-OVERLAY-NETWORK-CIDR
  <!-- code_snippet c2c_networking_stub_table_r3c1 end -->
    </code></pre></td>
    <td>
      <!-- code_snippet c2c_networking_stub_table_r3c2 start -->
      (Optional) Enter an IP range for the overlay network. The CIDR must specify an RFC 1918 range. If you do not set a custom range, the deployment uses <code>10.255.0.0/16</code>.
<br><br>See <a href="../../concepts/understand-cf-networking.html#app-comm">App Instance Communication</a> for more information.
      <!-- code_snippet c2c_networking_stub_table_r3c2 end -->
    </td>
  </tr>
  <tr>
    <td><pre><code>
    <!-- code_snippet c2c_networking_stub_table_r4c1 start -->
- type: replace
  path: /instance_groups/name=diego-cell/jobs/name=cni/properties/mtu?
  value: REPLACE-WITH-MTU
  <!-- code_snippet c2c_networking_stub_table_r4c1 end -->
    </code></pre></code>
    <td>
      <!-- code_snippet c2c_networking_stub_table_r4c2 start -->
      (Optional) You can manually configure the Maximum Transmission Unit (MTU) value to support additional encapsulation overhead.
      <!-- code_snippet c2c_networking_stub_table_r4c2 end -->
    </td>
  </tr>
</table>
